"""Tenable-provided templates: ``/editor/{type}/templates``."""

from __future__ import annotations

from fastapi.testclient import TestClient

from conftest import AUTH

LIST_KEYS = {
    "uuid", "name", "title", "desc", "order", "unsupported", "cloud_only",
    "subscription_only", "manager_only", "is_agent", "is_was",
    # Undocumented, but present on a live tenant.
    "icon",
}


def _templates(client: TestClient, template_type: str) -> list[dict]:
    response = client.get(f"/editor/{template_type}/templates", headers=AUTH)
    assert response.status_code == 200, response.text
    return response.json()["templates"]


def test_list_carries_every_documented_key(client: TestClient) -> None:
    templates = _templates(client, "scan")
    assert templates
    assert all(set(t) == LIST_KEYS for t in templates)


def test_uuids_keep_tenables_shape(client: TestClient) -> None:
    for template in _templates(client, "policy"):
        assert len(template["uuid"]) == 52


def test_scan_and_policy_lists_do_not_share_uuids(client: TestClient) -> None:
    scan = {t["name"]: t["uuid"] for t in _templates(client, "scan")}
    policy = {t["name"]: t["uuid"] for t in _templates(client, "policy")}
    shared = scan.keys() & policy.keys()
    assert "advanced" in shared
    assert all(scan[name] != policy[name] for name in shared)


def test_titles_repeat_between_network_and_agent_templates(client: TestClient) -> None:
    matches = [t for t in _templates(client, "scan") if t["title"] == "Malware Scan"]
    assert {t["name"] for t in matches} == {"malware", "agent_malware"}
    assert {t["is_agent"] for t in matches} == {None, True}


def test_non_agent_templates_report_null_not_false(client: TestClient) -> None:
    flags = {t["is_agent"] for t in _templates(client, "scan")}
    assert flags == {None, True}


def test_unknown_type_is_rejected(client: TestClient) -> None:
    response = client.get("/editor/bogus/templates", headers=AUTH)
    assert response.status_code == 400


def test_details_are_found_only_in_their_own_list(client: TestClient) -> None:
    advanced = next(t for t in _templates(client, "policy") if t["name"] == "advanced")

    found = client.get(f"/editor/policy/templates/{advanced['uuid']}", headers=AUTH)
    assert found.status_code == 200
    details = found.json()
    assert details["uuid"] == advanced["uuid"]
    assert set(details["settings"]) >= {"basic"}
    assert "families" in details["plugins"]

    assert client.get(f"/editor/scan/templates/{advanced['uuid']}", headers=AUTH).status_code == 404


def test_only_plugin_selecting_templates_carry_plugins(client: TestClient) -> None:
    basic = next(t for t in _templates(client, "policy") if t["name"] == "basic")
    details = client.get(f"/editor/policy/templates/{basic['uuid']}", headers=AUTH).json()
    assert "plugins" not in details


def test_templates_need_credentials(client: TestClient) -> None:
    assert client.get("/editor/scan/templates").status_code == 401
