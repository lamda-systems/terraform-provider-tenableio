"""The non-tag endpoints.

These are covered at the level that matters for a provider: the response
*shape* each verb returns, since several of them differ from one another in
ways that are easy to get wrong.
"""

from __future__ import annotations

from fastapi.testclient import TestClient

from conftest import AUTH

TEMPLATE = "ab4bacd2-05f6-425c-9d79-3ba7478a1d9a"


# -- folders ---------------------------------------------------------------


def test_create_folder_returns_only_the_id(client: TestClient) -> None:
    response = client.post("/folders", json={"name": "Terraform"}, headers=AUTH)
    assert response.status_code == 200
    assert set(response.json()) == {"id"}


def test_folders_are_seeded_with_the_system_pair(client: TestClient) -> None:
    folders = client.get("/folders", headers=AUTH).json()["folders"]
    assert {f["name"] for f in folders} == {"My Scans", "Trash"}
    assert all(f["custom"] == 0 for f in folders)


def test_system_folders_cannot_be_renamed_or_deleted(client: TestClient) -> None:
    folders = client.get("/folders", headers=AUTH).json()["folders"]
    main = next(f for f in folders if f["name"] == "My Scans")
    assert client.put(f"/folders/{main['id']}", json={"name": "x"}, headers=AUTH).status_code == 400
    assert client.delete(f"/folders/{main['id']}", headers=AUTH).status_code == 400


def test_folder_edit_and_delete_return_empty_bodies(client: TestClient) -> None:
    folder_id = client.post("/folders", json={"name": "Terraform"}, headers=AUTH).json()["id"]
    edited = client.put(f"/folders/{folder_id}", json={"name": "Renamed"}, headers=AUTH)
    assert edited.status_code == 200
    assert edited.content == b""

    folders = client.get("/folders", headers=AUTH).json()["folders"]
    assert any(f["name"] == "Renamed" for f in folders)
    assert client.delete(f"/folders/{folder_id}", headers=AUTH).status_code == 200


# -- networks --------------------------------------------------------------


def test_default_network_is_seeded_and_undeletable(client: TestClient) -> None:
    networks = client.get("/networks", headers=AUTH).json()["networks"]
    default = next(n for n in networks if n["is_default"])
    assert client.delete(f"/networks/{default['uuid']}", headers=AUTH).status_code == 400


def test_network_crud(client: TestClient) -> None:
    created = client.post(
        "/networks", json={"name": "Lab", "assets_ttl_days": 30}, headers=AUTH
    )
    assert created.status_code == 200
    network = created.json()
    assert network["assets_ttl_days"] == 30
    assert network["is_default"] is False

    assert client.get(f"/networks/{network['uuid']}", headers=AUTH).status_code == 200

    updated = client.put(
        f"/networks/{network['uuid']}",
        json={"name": "Lab", "description": "updated"},
        headers=AUTH,
    )
    assert updated.json()["description"] == "updated"
    assert client.delete(f"/networks/{network['uuid']}", headers=AUTH).status_code == 200


def test_network_ttl_is_range_checked(client: TestClient) -> None:
    for ttl in (13, 366):
        response = client.post(
            "/networks", json={"name": f"n{ttl}", "assets_ttl_days": ttl}, headers=AUTH
        )
        assert response.status_code == 400


def test_network_defaults_its_ttl_when_omitted(client: TestClient) -> None:
    response = client.post("/networks", json={"name": "Lab"}, headers=AUTH)
    assert response.json()["assets_ttl_days"] == 180


# -- exclusions ------------------------------------------------------------


def test_exclusion_crud(client: TestClient) -> None:
    created = client.post(
        "/exclusions",
        json={
            "name": "Maintenance",
            "members": "10.0.0.0/24",
            "schedule": {"enabled": True, "timezone": "Etc/UTC"},
        },
        headers=AUTH,
    )
    assert created.status_code == 200
    exclusion = created.json()
    assert exclusion["schedule"]["enabled"] is True
    assert exclusion["schedule"]["timezone"] == "Etc/UTC"

    assert client.get(f"/exclusions/{exclusion['id']}", headers=AUTH).status_code == 200
    updated = client.put(
        f"/exclusions/{exclusion['id']}",
        json={"name": "Maintenance", "members": "10.0.1.0/24"},
        headers=AUTH,
    )
    assert updated.json()["members"] == "10.0.1.0/24"
    assert client.delete(f"/exclusions/{exclusion['id']}", headers=AUTH).status_code == 200


def test_exclusion_without_a_schedule_still_reports_one(client: TestClient) -> None:
    response = client.post(
        "/exclusions", json={"name": "One-off", "members": "10.0.0.1"}, headers=AUTH
    )
    assert response.json()["schedule"] == {"enabled": False}


def test_exclusion_rejects_an_unknown_network(client: TestClient) -> None:
    response = client.post(
        "/exclusions",
        json={"name": "x", "members": "10.0.0.1", "network_id": "nope"},
        headers=AUTH,
    )
    assert response.status_code == 400


# -- agent groups ----------------------------------------------------------


def test_agent_group_crud(client: TestClient) -> None:
    created = client.post("/scanners/null/agent-groups", json={"name": "Linux"}, headers=AUTH)
    assert created.status_code == 200
    group = created.json()

    assert client.get(f"/scanners/null/agent-groups/{group['id']}", headers=AUTH).status_code == 200
    assert client.get("/scanners/null/agent-groups", headers=AUTH).json()["groups"]

    duplicate = client.post("/scanners/null/agent-groups", json={"name": "Linux"}, headers=AUTH)
    assert duplicate.status_code == 400

    assert client.delete(f"/scanners/null/agent-groups/{group['id']}", headers=AUTH).status_code == 200


def test_agent_group_path_does_not_shadow_scanner_lookup(client: TestClient) -> None:
    """/scanners/null/agent-groups and /scanners/{id} must not collide."""
    assert client.get("/scanners/null/agent-groups", headers=AUTH).status_code == 200
    assert client.get("/scanners/100001", headers=AUTH).status_code == 200


# -- scanners --------------------------------------------------------------


def test_scanners_are_seeded_and_read_only(client: TestClient) -> None:
    scanners = client.get("/scanners", headers=AUTH).json()["scanners"]
    assert len(scanners) == 2
    assert client.get("/scanners/100001", headers=AUTH).json()["name"] == "US Cloud Scanner"
    assert client.get("/scanners/999", headers=AUTH).status_code == 404


# -- policies --------------------------------------------------------------


def test_policy_create_returns_the_narrow_shape(client: TestClient) -> None:
    response = client.post(
        "/policies",
        json={"uuid": TEMPLATE, "settings": {"name": "Baseline"}},
        headers=AUTH,
    )
    assert response.status_code == 200
    assert set(response.json()) == {"policy_id", "policy_name"}


def test_policy_detail_is_flat_and_keeps_the_template(client: TestClient) -> None:
    policy_id = client.post(
        "/policies",
        json={"uuid": TEMPLATE, "settings": {"name": "Baseline", "visibility": "shared"}},
        headers=AUTH,
    ).json()["policy_id"]

    detail = client.get(f"/policies/{policy_id}", headers=AUTH).json()
    assert detail["name"] == "Baseline"
    assert detail["visibility"] == "shared"
    assert detail["template_uuid"] == TEMPLATE
    # The policy's own uuid is distinct from the template it derives from.
    assert detail["uuid"] != TEMPLATE


def test_policy_update_returns_an_empty_body(client: TestClient) -> None:
    policy_id = client.post(
        "/policies", json={"uuid": TEMPLATE, "settings": {"name": "Baseline"}}, headers=AUTH
    ).json()["policy_id"]

    updated = client.put(
        f"/policies/{policy_id}", json={"settings": {"name": "Renamed"}}, headers=AUTH
    )
    assert updated.status_code == 200
    assert updated.content == b""
    assert client.get(f"/policies/{policy_id}", headers=AUTH).json()["name"] == "Renamed"


def test_policy_visibility_is_validated(client: TestClient) -> None:
    response = client.post(
        "/policies",
        json={"uuid": TEMPLATE, "settings": {"name": "x", "visibility": "everyone"}},
        headers=AUTH,
    )
    assert response.status_code == 400


# -- scans -----------------------------------------------------------------


def make_scan(client: TestClient, **settings) -> dict:
    payload = {"name": "Nightly", "text_targets": "10.0.0.0/24", "enabled": True}
    payload.update(settings)
    response = client.post("/scans", json={"uuid": TEMPLATE, "settings": payload}, headers=AUTH)
    assert response.status_code == 200, response.text
    return response.json()["scan"]


def test_scan_create_wraps_the_detail_shape(client: TestClient) -> None:
    scan = make_scan(client)
    assert scan["text_targets"] == "10.0.0.0/24"
    assert scan["emails"] == ""
    assert scan["id"]


def test_scan_get_returns_the_renamed_info_shape(client: TestClient) -> None:
    """GET renames half the keys relative to POST. Both must be reproduced."""
    scan = make_scan(client, emails="ops@example.com")
    info = client.get(f"/scans/{scan['id']}", headers=AUTH).json()["info"]

    assert info["object_id"] == scan["id"]
    assert info["targets"] == "10.0.0.0/24"
    assert info["notification_email_address"] == "ops@example.com"
    assert info["scan_type"] == scan["type"]
    assert info["scanner_name"] == TEMPLATE
    # The POST spellings must be absent, or a provider could read the wrong key.
    assert "id" not in info
    assert "text_targets" not in info
    assert "emails" not in info


def test_scan_list_filters_by_folder(client: TestClient) -> None:
    folder_id = client.post("/folders", json={"name": "Terraform"}, headers=AUTH).json()["id"]
    make_scan(client, name="In folder", folder_id=folder_id)
    make_scan(client, name="Elsewhere")

    everything = client.get("/scans", headers=AUTH).json()["scans"]
    assert len(everything) == 2

    filtered = client.get(f"/scans?folder_id={folder_id}", headers=AUTH).json()["scans"]
    assert [s["name"] for s in filtered] == ["In folder"]


def test_scan_update_returns_an_empty_body(client: TestClient) -> None:
    scan = make_scan(client)
    updated = client.put(
        f"/scans/{scan['id']}",
        json={"settings": {"name": "Renamed", "enabled": False}},
        headers=AUTH,
    )
    assert updated.status_code == 200
    assert updated.content == b""
    assert client.get(f"/scans/{scan['id']}", headers=AUTH).json()["info"]["name"] == "Renamed"


def test_scan_rejects_dangling_references(client: TestClient) -> None:
    for field, value in (("folder_id", 999), ("scanner_id", 999), ("policy_id", 999)):
        response = client.post(
            "/scans",
            json={"uuid": TEMPLATE, "settings": {"name": "x", field: value}},
            headers=AUTH,
        )
        assert response.status_code == 400, field
        assert response.json()["code"] == "not_found"


def test_scan_launch_is_validated(client: TestClient) -> None:
    response = client.post(
        "/scans",
        json={"uuid": TEMPLATE, "settings": {"name": "x", "launch": "HOURLY"}},
        headers=AUTH,
    )
    assert response.status_code == 400


def test_scan_delete(client: TestClient) -> None:
    scan = make_scan(client)
    assert client.delete(f"/scans/{scan['id']}", headers=AUTH).status_code == 200
    assert client.get(f"/scans/{scan['id']}", headers=AUTH).status_code == 404


# -- workbenches -----------------------------------------------------------


def test_assets_are_seeded(client: TestClient) -> None:
    assets = client.get("/workbenches/assets", headers=AUTH).json()["assets"]
    assert len(assets) == 2


def test_asset_info_is_wrapped(client: TestClient) -> None:
    asset_id = client.get("/workbenches/assets", headers=AUTH).json()["assets"][0]["id"]
    info = client.get(f"/workbenches/assets/{asset_id}/info", headers=AUTH).json()["info"]
    assert info["id"] == asset_id
    assert "counts" in info


def test_asset_filters_are_applied(client: TestClient) -> None:
    response = client.get(
        "/workbenches/assets?filter.0.filter=ipv4&filter.0.quality=eq&filter.0.value=10.0.1.10",
        headers=AUTH,
    )
    assets = response.json()["assets"]
    assert [a["ipv4"] for a in assets] == [["10.0.1.10"]]


def test_unknown_asset_is_404(client: TestClient) -> None:
    assert client.get("/workbenches/assets/nope/info", headers=AUTH).status_code == 404


# -- asset tag filter catalogue --------------------------------------------


def test_asset_tag_filters_expose_operators_and_controls(client: TestClient) -> None:
    filters = client.get("/tags/assets/filters", headers=AUTH).json()["filters"]
    by_name = {f["name"]: f for f in filters}

    assert "eq" in by_name["ipv4"]["operators"]
    assert by_name["ipv4"]["control"]["type"] == "entry"
    assert by_name["ipv4"]["control"]["regex"]

    dropdown = by_name["asset_class"]["control"]
    assert dropdown["type"] == "dropdown"
    # Options are {name, value} objects, not bare strings.
    assert all(set(entry) == {"name", "value"} for entry in dropdown["list"])
