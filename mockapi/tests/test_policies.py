"""User-defined scan templates: settings, plugin families and the details shape."""

from __future__ import annotations

from fastapi.testclient import TestClient

from conftest import AUTH, make_client
from tenableio_mock.config import PolicyDetailsSettings


def _template(client: TestClient, name: str, kind: str = "policy") -> str:
    templates = client.get(f"/editor/{kind}/templates", headers=AUTH).json()["templates"]
    return next(t["uuid"] for t in templates if t["name"] == name)


def _create(client: TestClient, template: str = "advanced", **body) -> int:
    payload = {"uuid": _template(client, template), "settings": {"name": "Baseline"}}
    for key, value in body.items():
        if key == "settings":
            payload["settings"].update(value)
        else:
            payload[key] = value
    response = client.post("/policies", json=payload, headers=AUTH)
    assert response.status_code == 200, response.text
    return response.json()["policy_id"]


def _details(client: TestClient, policy_id: int) -> dict:
    response = client.get(f"/policies/{policy_id}", headers=AUTH)
    assert response.status_code == 200, response.text
    return response.json()


def test_details_nest_the_settings_under_the_template_uuid(client: TestClient) -> None:
    policy_id = _create(client, settings={"description": "weekly"})
    details = _details(client, policy_id)

    assert set(details) == {"uuid", "settings", "plugins"}
    assert details["uuid"] == _template(client, "advanced")
    assert details["settings"]["name"] == "Baseline"
    assert details["settings"]["description"] == "weekly"
    # No metadata: that is the list's job.
    assert "owner" not in details and "id" not in details


def test_settings_come_back_as_strings_apart_from_two_integers(client: TestClient) -> None:
    settings = _details(client, _create(client))["settings"]
    assert settings["ping_the_remote_host"] == "yes"
    assert settings["max_compliance_check_timeout"] == 300
    assert settings["http_reauth_delay"] == 0
    assert all(
        isinstance(v, str) for k, v in settings.items()
        if k not in ("max_compliance_check_timeout", "http_reauth_delay")
    )


def test_create_stores_settings_and_integers_as_strings(client: TestClient) -> None:
    policy_id = _create(client, "agent_advanced", settings={"refresh_reporting_frequency_days": 14})
    assert _details(client, policy_id)["settings"]["refresh_reporting_frequency_days"] == "14"


def test_an_unknown_template_is_rejected(client: TestClient) -> None:
    response = client.post(
        "/policies",
        json={"uuid": _template(client, "advanced", kind="scan"), "settings": {"name": "x"}},
        headers=AUTH,
    )
    assert response.status_code == 400
    assert "policy template list" in response.json()["message"]


def test_settings_the_template_does_not_define_are_discarded(client: TestClient) -> None:
    policy_id = _create(client, "agent_basic", settings={"ping_the_remote_host": "no", "made_up": "x"})
    settings = _details(client, policy_id)["settings"]
    assert "ping_the_remote_host" not in settings
    assert "made_up" not in settings


def test_update_preserves_omitted_settings(client: TestClient) -> None:
    policy_id = _create(client, settings={"udp_ping": "yes", "description": "kept"})
    client.put(f"/policies/{policy_id}", json={"settings": {"tcp_ping": "no"}}, headers=AUTH)

    settings = _details(client, policy_id)["settings"]
    assert settings["udp_ping"] == "yes"
    assert settings["tcp_ping"] == "no"
    assert settings["description"] == "kept"


def test_update_ignores_a_new_template_uuid(client: TestClient) -> None:
    policy_id = _create(client)
    client.put(
        f"/policies/{policy_id}",
        json={"uuid": _template(client, "basic"), "settings": {"name": "Baseline"}},
        headers=AUTH,
    )
    assert _details(client, policy_id)["uuid"] == _template(client, "advanced")


def test_dependent_settings_are_omitted_while_their_toggle_is_off(client: TestClient) -> None:
    policy_id = _create(client, settings={"webcrawl_max_depth": "4"})
    assert "webcrawl_max_depth" not in _details(client, policy_id)["settings"]

    client.put(f"/policies/{policy_id}", json={"settings": {"scan_webapps": "yes"}}, headers=AUTH)
    assert _details(client, policy_id)["settings"]["webcrawl_max_depth"] == "4"


def test_dependent_settings_are_all_reported_under_the_quirk() -> None:
    client = make_client(policy_details_settings=PolicyDetailsSettings.ALL)
    policy_id = _create(client, settings={"webcrawl_max_depth": "4"})
    assert _details(client, policy_id)["settings"]["webcrawl_max_depth"] == "4"


def test_plugin_families_are_merged(client: TestClient) -> None:
    policy_id = _create(client, plugins={"Backdoors": {"status": "disabled"}})
    client.put(
        f"/policies/{policy_id}", json={"plugins": {"SCADA": {"status": "disabled"}}}, headers=AUTH
    )
    plugins = _details(client, policy_id)["plugins"]
    assert plugins["Backdoors"] == {"status": "disabled"}
    assert plugins["SCADA"] == {"status": "disabled"}
    assert plugins["Web Servers"] == {"status": "enabled"}


def test_plugins_need_a_template_with_plugin_selection(client: TestClient) -> None:
    response = client.post(
        "/policies",
        json={
            "uuid": _template(client, "basic"),
            "settings": {"name": "x"},
            "plugins": {"Backdoors": {"status": "disabled"}},
        },
        headers=AUTH,
    )
    assert response.status_code == 400
    assert client.get("/policies", headers=AUTH).json()["policies"] == []


def test_plugin_family_and_status_are_validated(client: TestClient) -> None:
    template = _template(client, "advanced")
    for plugins in ({"Nope": {"status": "enabled"}}, {"Backdoors": {"status": "mixed"}}):
        response = client.post(
            "/policies", json={"uuid": template, "settings": {"name": "x"}, "plugins": plugins}, headers=AUTH
        )
        assert response.status_code == 400, plugins


def test_templates_without_plugin_selection_report_no_plugins(client: TestClient) -> None:
    assert "plugins" not in _details(client, _create(client, "basic"))
