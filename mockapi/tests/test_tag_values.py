"""Tag values, including the request/response shape divergence for filters."""

from __future__ import annotations

import json

import pytest
from fastapi.testclient import TestClient

from tenableio_mock.config import OnOmit

from conftest import AUTH, make_client

RULE = {"property": "operating_system", "operator": "equals", "value": ["FreeBSD"]}
FILTERS = {"asset": {"and": [RULE]}}


def create_value(client: TestClient, category: dict, **overrides) -> dict:
    payload = {"category_uuid": category["uuid"], "value": "London"}
    payload.update(overrides)
    response = client.post("/tags/values", json=payload, headers=AUTH)
    assert response.status_code == 200, response.text
    return response.json()


# -- category resolution ---------------------------------------------------


def test_category_uuid_binds_to_an_existing_category(
    client: TestClient, category: dict
) -> None:
    value = create_value(client, category)
    assert value["category_uuid"] == category["uuid"]
    assert value["category_name"] == "Location"


def test_unknown_category_uuid_is_rejected(client: TestClient) -> None:
    response = client.post(
        "/tags/values", json={"category_uuid": "nope", "value": "London"}, headers=AUTH
    )
    assert response.status_code == 400
    assert response.json()["code"] == "not_found"


def test_category_name_creates_the_category_on_demand(client: TestClient) -> None:
    """The asymmetry with POST /tags/categories is real, not an oversight.

    Creating a category directly rejects a duplicate name; creating a *value*
    with a category_name happily reuses or creates one.
    """
    response = client.post(
        "/tags/values",
        json={
            "category_name": "Environment",
            "category_description": "Deployment tier",
            "value": "prod",
        },
        headers=AUTH,
    )
    assert response.status_code == 200
    body = response.json()
    assert body["category_name"] == "Environment"
    assert body["category_description"] == "Deployment tier"

    categories = client.get("/tags/categories", headers=AUTH).json()["categories"]
    assert [c["name"] for c in categories] == ["Environment"]


def test_category_name_reuses_an_existing_category(
    client: TestClient, category: dict
) -> None:
    value = client.post(
        "/tags/values", json={"category_name": "Location", "value": "Paris"}, headers=AUTH
    ).json()
    assert value["category_uuid"] == category["uuid"]
    assert len(client.get("/tags/categories", headers=AUTH).json()["categories"]) == 1


@pytest.mark.parametrize(
    "payload",
    [
        {"value": "London"},
        {"value": "London", "category_uuid": "u", "category_name": "n"},
    ],
)
def test_exactly_one_category_selector_is_required(
    client: TestClient, payload: dict
) -> None:
    response = client.post("/tags/values", json=payload, headers=AUTH)
    assert response.status_code == 400


def test_duplicate_value_within_a_category_is_rejected(
    client: TestClient, category: dict
) -> None:
    create_value(client, category)
    response = client.post(
        "/tags/values",
        json={"category_uuid": category["uuid"], "value": "London"},
        headers=AUTH,
    )
    assert response.status_code == 400
    assert response.json()["code"] == "duplicate"


def test_the_same_value_in_another_category_is_fine(
    client: TestClient, category: dict
) -> None:
    """Uniqueness is on the (category, value) pair, not the value alone."""
    create_value(client, category)
    other = client.post("/tags/categories", json={"name": "Region"}, headers=AUTH).json()
    assert create_value(client, other)["value"] == "London"


# -- static vs dynamic -----------------------------------------------------


def test_a_value_without_filters_is_static(client: TestClient, category: dict) -> None:
    value = create_value(client, category)
    assert value["type"] == "static"
    assert "filters" not in value


def test_filters_make_the_tag_dynamic(client: TestClient, category: dict) -> None:
    value = create_value(client, category, filters=FILTERS)
    assert value["type"] == "dynamic"


def test_response_filters_are_a_json_string_not_an_object(
    client: TestClient, category: dict
) -> None:
    """The single most important divergence the mock reproduces."""
    value = create_value(client, category, filters=FILTERS)
    asset = value["filters"]["asset"]
    assert isinstance(asset, str)

    decoded = json.loads(asset)
    assert decoded == {
        "and": [{"field": "operating_system", "operator": "eq", "value": "FreeBSD"}]
    }


def test_response_renames_property_to_field_and_shortens_the_operator(
    client: TestClient, category: dict
) -> None:
    value = create_value(client, category, filters=FILTERS)
    rule = json.loads(value["filters"]["asset"])["and"][0]
    assert "property" not in rule
    assert rule["field"] == "operating_system"
    assert rule["operator"] == "eq"


def test_a_single_value_collapses_to_a_bare_string(
    client: TestClient, category: dict
) -> None:
    value = create_value(client, category, filters=FILTERS)
    assert json.loads(value["filters"]["asset"])["and"][0]["value"] == "FreeBSD"


def test_multiple_values_stay_an_array(client: TestClient, category: dict) -> None:
    filters = {
        "asset": {
            "and": [
                {"property": "ipv4", "operator": "eq", "value": ["10.0.0.1", "10.0.0.2"]}
            ]
        }
    }
    value = create_value(client, category, filters=filters)
    assert json.loads(value["filters"]["asset"])["and"][0]["value"] == [
        "10.0.0.1",
        "10.0.0.2",
    ]


def test_the_encoded_filter_is_byte_stable_across_reads(
    client: TestClient, category: dict
) -> None:
    """Terraform compares state verbatim, so re-encoding must not shift bytes."""
    value = create_value(client, category, filters=FILTERS)
    first = client.get(f"/tags/values/{value['uuid']}", headers=AUTH).json()
    second = client.get(f"/tags/values/{value['uuid']}", headers=AUTH).json()
    assert first["filters"]["asset"] == second["filters"]["asset"] == value["filters"]["asset"]


def test_or_rules_are_supported(client: TestClient, category: dict) -> None:
    filters = {"asset": {"or": [RULE]}}
    value = create_value(client, category, filters=filters)
    assert "or" in json.loads(value["filters"]["asset"])


# -- update ----------------------------------------------------------------


def test_update_changes_the_value(client: TestClient, category: dict) -> None:
    value = create_value(client, category)
    response = client.put(
        f"/tags/values/{value['uuid']}", json={"value": "Paris"}, headers=AUTH
    )
    assert response.status_code == 200
    assert response.json()["value"] == "Paris"


def test_update_can_turn_a_static_tag_dynamic(client: TestClient, category: dict) -> None:
    value = create_value(client, category)
    response = client.put(
        f"/tags/values/{value['uuid']}",
        json={"value": "London", "filters": FILTERS},
        headers=AUTH,
    )
    assert response.json()["type"] == "dynamic"


def test_omitted_filters_clear_by_default(client: TestClient, category: dict) -> None:
    value = create_value(client, category, filters=FILTERS)
    response = client.put(
        f"/tags/values/{value['uuid']}", json={"value": "London"}, headers=AUTH
    )
    assert response.json()["type"] == "static"
    assert "filters" not in response.json()


def test_omitted_filters_preserve_under_the_quirk(category: dict) -> None:
    """Why the provider forces replacement instead of updating.

    The docs never resolve this, so the provider cannot rely on either
    behaviour -- both are reachable here.
    """
    c = make_client(on_omitted_filters=OnOmit.PRESERVES)
    cat = c.post("/tags/categories", json={"name": "Location"}, headers=AUTH).json()
    value = c.post(
        "/tags/values",
        json={"category_uuid": cat["uuid"], "value": "London", "filters": FILTERS},
        headers=AUTH,
    ).json()
    response = c.put(f"/tags/values/{value['uuid']}", json={"value": "London"}, headers=AUTH)
    assert response.json()["type"] == "dynamic"
    assert response.json()["filters"]["asset"] == value["filters"]["asset"]


def test_update_unknown_is_404(client: TestClient) -> None:
    assert client.put("/tags/values/nope", json={"value": "x"}, headers=AUTH).status_code == 404


# -- read and delete -------------------------------------------------------


def test_list_and_get(client: TestClient, category: dict) -> None:
    value = create_value(client, category)
    listed = client.get("/tags/values", headers=AUTH).json()
    assert [v["uuid"] for v in listed["values"]] == [value["uuid"]]
    assert client.get(f"/tags/values/{value['uuid']}", headers=AUTH).status_code == 200


def test_category_rename_shows_up_in_its_values(client: TestClient, category: dict) -> None:
    value = create_value(client, category)
    client.put(f"/tags/categories/{category['uuid']}", json={"name": "Region"}, headers=AUTH)
    refetched = client.get(f"/tags/values/{value['uuid']}", headers=AUTH).json()
    assert refetched["category_name"] == "Region"


def test_delete(client: TestClient, category: dict) -> None:
    value = create_value(client, category)
    assert client.delete(f"/tags/values/{value['uuid']}", headers=AUTH).status_code == 200
    assert client.get(f"/tags/values/{value['uuid']}", headers=AUTH).status_code == 404
    assert client.delete(f"/tags/values/{value['uuid']}", headers=AUTH).status_code == 404


def test_access_control_block_is_present(client: TestClient, category: dict) -> None:
    """Present so the mock's responses are not structurally narrower than production's."""
    value = create_value(client, category)
    assert "current_user_permissions" in value["access_control"]
