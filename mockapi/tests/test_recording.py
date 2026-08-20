"""The request log.

This is how a test asserts on what a provider *actually put on the wire*, as
opposed to what it meant to send. Checking that a key is absent is what catches
a serialiser that silently drops a meaningful empty value -- Go's ``omitempty``
being the case that prompted all this.
"""

from __future__ import annotations

from fastapi.testclient import TestClient

from conftest import AUTH


def test_requests_are_recorded_in_order(client: TestClient) -> None:
    client.post("/tags/categories", json={"name": "Location"}, headers=AUTH)
    client.get("/tags/categories", headers=AUTH)

    records = client.get("/__mock/requests").json()["requests"]
    assert [(r["method"], r["path"]) for r in records] == [
        ("POST", "/tags/categories"),
        ("GET", "/tags/categories"),
    ]
    assert records[0]["status"] == 200


def test_body_keys_distinguish_absent_from_empty(client: TestClient) -> None:
    """The distinction the whole design turns on.

    A body carrying ``description: ""`` is an instruction to clear. A body with
    no ``description`` key at all is silence. They are different requests, and
    only ``body_keys`` tells them apart -- inspecting values cannot.
    """
    client.post("/tags/categories", json={"name": "Explicit", "description": ""}, headers=AUTH)
    client.post("/tags/categories", json={"name": "Silent"}, headers=AUTH)

    records = client.get("/__mock/requests?method=POST").json()["requests"]
    explicit, silent = records[0], records[1]

    assert "description" in explicit["body_keys"]
    assert explicit["body"]["description"] == ""

    assert "description" not in silent["body_keys"]


def test_failed_requests_are_recorded_too(client: TestClient) -> None:
    """A 400 or 401 is often the thing under test, so it must show up."""
    client.get("/tags/categories")  # no auth
    client.post("/tags/categories", json={}, headers=AUTH)  # invalid

    records = client.get("/__mock/requests").json()["requests"]
    assert [r["status"] for r in records] == [401, 400]


def test_query_strings_are_recorded(client: TestClient) -> None:
    client.get("/scans?folder_id=3", headers=AUTH)
    record = client.get("/__mock/requests").json()["requests"][0]
    assert record["query"] == "folder_id=3"


def test_admin_traffic_is_not_recorded(client: TestClient) -> None:
    """Otherwise reading the log would append to the log."""
    client.get("/tags/categories", headers=AUTH)
    client.get("/__mock/requests")
    client.get("/__mock/health")

    records = client.get("/__mock/requests").json()["requests"]
    assert len(records) == 1
    assert records[0]["path"] == "/tags/categories"


def test_records_can_be_filtered(client: TestClient) -> None:
    client.post("/tags/categories", json={"name": "A"}, headers=AUTH)
    client.post("/tags/categories", json={"name": "B"}, headers=AUTH)
    client.get("/tags/categories", headers=AUTH)

    assert client.get("/__mock/requests?method=POST").json()["count"] == 2
    assert client.get("/__mock/requests?path=/tags/categories").json()["count"] == 3
    assert client.get("/__mock/requests?method=GET&path=/folders").json()["count"] == 0


def test_reset_clears_objects_and_reseeds(client: TestClient) -> None:
    client.post("/tags/categories", json={"name": "Location"}, headers=AUTH)
    assert client.post("/__mock/reset").status_code == 200

    assert client.get("/tags/categories", headers=AUTH).json()["categories"] == []
    assert client.get("/__mock/requests").json()["count"] == 1  # only the GET above
    # Seeded read-only data comes back.
    assert len(client.get("/scanners", headers=AUTH).json()["scanners"]) == 2


def test_reset_requests_only_keeps_objects(client: TestClient) -> None:
    client.post("/tags/categories", json={"name": "Location"}, headers=AUTH)
    client.post("/__mock/reset?requests_only=true")

    assert client.get("/__mock/requests").json()["count"] == 0
    assert len(client.get("/tags/categories", headers=AUTH).json()["categories"]) == 1


def test_settings_endpoint_reports_the_active_quirks(client: TestClient) -> None:
    """So a failing pipeline can print the configuration it ran against."""
    body = client.get("/__mock/settings").json()
    assert body["quirks"]["on_omitted_description"] == "clears"
    assert body["quirks"]["on_omitted_filters"] == "clears"
    assert body["quirks"]["lowercase_category_names"] is False
    assert body["seed"] is True


def test_non_json_bodies_do_not_break_recording(client: TestClient) -> None:
    client.post(
        "/tags/categories",
        content=b"not json",
        headers={**AUTH, "Content-Type": "application/json"},
    )
    record = client.get("/__mock/requests").json()["requests"][0]
    assert record["body"] is None
    assert record["body_keys"] == []
