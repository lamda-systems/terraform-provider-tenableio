"""Tag categories.

The duplicate-name case is the one that matters most. Tenable.io answers 400;
it does not return the existing category. A mock that upserted here would let a
provider ship a bug where the planned ``name`` and the applied ``name`` differ,
which Terraform aborts on with "Provider produced inconsistent result after
apply".
"""

from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from tenableio_mock.config import OnOmit

from conftest import AUTH, make_client


def test_create_returns_the_whole_category(client: TestClient) -> None:
    response = client.post(
        "/tags/categories",
        json={"name": "Location", "description": "Where the asset lives"},
        headers=AUTH,
    )
    assert response.status_code == 200
    body = response.json()
    assert body["name"] == "Location"
    assert body["description"] == "Where the asset lives"
    assert body["reserved"] is False
    assert body["created_by"] == "terraform@example.com"
    assert body["created_at"] == body["updated_at"]


def test_create_echoes_the_name_verbatim(client: TestClient) -> None:
    """No normalisation by default -- not even case folding.

    ``name`` is a required attribute on the provider's resource, so its planned
    value is always known. Anything but a verbatim echo breaks the apply.
    """
    response = client.post("/tags/categories", json={"name": "MiXeD CaSe"}, headers=AUTH)
    assert response.json()["name"] == "MiXeD CaSe"


def test_omitted_description_defaults_to_empty_string(client: TestClient) -> None:
    response = client.post("/tags/categories", json={"name": "Location"}, headers=AUTH)
    assert response.json()["description"] == ""


def test_duplicate_name_is_rejected_not_upserted(client: TestClient, category: dict) -> None:
    response = client.post("/tags/categories", json={"name": "Location"}, headers=AUTH)
    assert response.status_code == 400
    body = response.json()
    assert body["message"] == "A category with the name you specified already exists."
    assert body["code"] == "duplicate"


@pytest.mark.parametrize(
    ("payload", "fragment"),
    [
        ({"name": ""}, "required"),
        ({"name": "   "}, "required"),
        ({"name": "x" * 128}, "127"),
        ({"name": "has:colon"}, "colon"),
        ({"name": "ok", "description": "d" * 3001}, "3000"),
    ],
)
def test_validation(client: TestClient, payload: dict, fragment: str) -> None:
    response = client.post("/tags/categories", json=payload, headers=AUTH)
    assert response.status_code == 400
    assert fragment in response.json()["message"]


def test_get_and_list(client: TestClient, category: dict) -> None:
    fetched = client.get(f"/tags/categories/{category['uuid']}", headers=AUTH)
    assert fetched.status_code == 200
    assert fetched.json()["uuid"] == category["uuid"]

    listed = client.get("/tags/categories", headers=AUTH)
    assert listed.status_code == 200
    body = listed.json()
    assert [c["uuid"] for c in body["categories"]] == [category["uuid"]]
    assert body["pagination"]["total"] == 1


def test_get_unknown_is_404(client: TestClient) -> None:
    assert client.get("/tags/categories/nope", headers=AUTH).status_code == 404


def test_update_replaces_name_and_description(client: TestClient, category: dict) -> None:
    response = client.put(
        f"/tags/categories/{category['uuid']}",
        json={"name": "Region", "description": "renamed"},
        headers=AUTH,
    )
    assert response.status_code == 200
    assert response.json()["name"] == "Region"
    assert response.json()["description"] == "renamed"


def test_update_to_a_taken_name_is_rejected(client: TestClient, category: dict) -> None:
    other = client.post("/tags/categories", json={"name": "Region"}, headers=AUTH).json()
    response = client.put(
        f"/tags/categories/{other['uuid']}", json={"name": "Location"}, headers=AUTH
    )
    assert response.status_code == 400
    assert response.json()["code"] == "duplicate"


def test_update_keeping_its_own_name_is_allowed(client: TestClient, category: dict) -> None:
    response = client.put(
        f"/tags/categories/{category['uuid']}",
        json={"name": "Location", "description": "same name, new text"},
        headers=AUTH,
    )
    assert response.status_code == 200


def test_explicit_empty_description_always_clears(client: TestClient) -> None:
    """An empty string on the wire is an instruction, under either quirk.

    Only *absence* is ambiguous. This is why a provider should send the field
    rather than eliding it.
    """
    for policy in (OnOmit.CLEARS, OnOmit.PRESERVES):
        c = make_client(on_omitted_description=policy)
        created = c.post(
            "/tags/categories",
            json={"name": "Location", "description": "original"},
            headers=AUTH,
        ).json()
        updated = c.put(
            f"/tags/categories/{created['uuid']}",
            json={"name": "Location", "description": ""},
            headers=AUTH,
        )
        assert updated.json()["description"] == "", policy


def test_omitted_description_clears_by_default() -> None:
    c = make_client()
    created = c.post(
        "/tags/categories",
        json={"name": "Location", "description": "original"},
        headers=AUTH,
    ).json()
    updated = c.put(
        f"/tags/categories/{created['uuid']}", json={"name": "Location"}, headers=AUTH
    )
    assert updated.json()["description"] == ""


def test_omitted_description_preserves_under_the_quirk() -> None:
    """The configuration that reproduces the reported apply failure.

    A provider that plans ``description = ""`` but drops the key from the body
    gets the old text echoed back here, and Terraform rejects the result.
    """
    c = make_client(on_omitted_description=OnOmit.PRESERVES)
    created = c.post(
        "/tags/categories",
        json={"name": "Location", "description": "The geographic location of the asset"},
        headers=AUTH,
    ).json()
    updated = c.put(
        f"/tags/categories/{created['uuid']}", json={"name": "Location"}, headers=AUTH
    )
    assert updated.json()["description"] == "The geographic location of the asset"


def test_lowercase_quirk_folds_the_name() -> None:
    c = make_client(lowercase_category_names=True)
    response = c.post("/tags/categories", json={"name": "Location"}, headers=AUTH)
    assert response.json()["name"] == "location"


def test_delete_takes_the_categorys_values_with_it(client: TestClient, category: dict) -> None:
    value = client.post(
        "/tags/values",
        json={"category_uuid": category["uuid"], "value": "London"},
        headers=AUTH,
    ).json()

    assert client.delete(f"/tags/categories/{category['uuid']}", headers=AUTH).status_code == 200
    assert client.get(f"/tags/categories/{category['uuid']}", headers=AUTH).status_code == 404
    assert client.get(f"/tags/values/{value['uuid']}", headers=AUTH).status_code == 404


def test_delete_unknown_is_404(client: TestClient) -> None:
    assert client.delete("/tags/categories/nope", headers=AUTH).status_code == 404


def test_reject_unknown_fields_quirk() -> None:
    strict = make_client(reject_unknown_fields=True)
    response = strict.post(
        "/tags/categories", json={"name": "Location", "colour": "blue"}, headers=AUTH
    )
    assert response.status_code == 400
    assert "colour" in response.json()["message"]

    # Off by default: production ignores unknown fields.
    lenient = make_client()
    assert (
        lenient.post(
            "/tags/categories", json={"name": "Location", "colour": "blue"}, headers=AUTH
        ).status_code
        == 200
    )
