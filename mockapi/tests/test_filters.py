"""Filter encoding rules, exercised directly rather than through HTTP."""

from __future__ import annotations

import json

import pytest

from tenableio_mock.errors import TenableError
from tenableio_mock.filters import (
    MAX_RULES_PER_TAG,
    MAX_VALUES_PER_RULE,
    encode_asset_filter,
    normalise_operator,
)


@pytest.mark.parametrize(
    ("given", "expected"),
    [
        ("eq", "eq"),
        ("neq", "neq"),
        ("match", "match"),
        ("set-hasonly", "set-hasonly"),
        ("equals", "eq"),
        ("EQUALS", "eq"),
        ("does not equal", "neq"),
        ("contains", "match"),
        ("does not contain", "nmatch"),
        ("wildcard", "wc"),
    ],
)
def test_operators_accept_both_spellings_and_emit_the_short_code(
    given: str, expected: str
) -> None:
    assert normalise_operator(given) == expected


def test_unknown_operator_is_rejected() -> None:
    """The operator vocabulary is closed and documented, so a typo is an error."""
    with pytest.raises(TenableError) as excinfo:
        normalise_operator("sort-of-equals")
    assert excinfo.value.status_code == 400


def test_property_is_not_validated() -> None:
    """Unlike operators, the taggable-attribute set is open-ended.

    It depends on which connectors a tenant has configured, so the mock has no
    basis on which to reject one.
    """
    encoded = encode_asset_filter(
        {"and": [{"property": "some_connector_field", "operator": "eq", "value": "x"}]}
    )
    assert json.loads(encoded)["and"][0]["field"] == "some_connector_field"


def test_field_is_accepted_as_well_as_property() -> None:
    """So a caller round-tripping a previous response is not punished for it."""
    encoded = encode_asset_filter(
        {"and": [{"field": "ipv4", "operator": "eq", "value": "10.0.0.1"}]}
    )
    assert json.loads(encoded)["and"][0]["field"] == "ipv4"


def test_a_bare_string_value_is_accepted() -> None:
    encoded = encode_asset_filter(
        {"and": [{"property": "ipv4", "operator": "eq", "value": "10.0.0.1"}]}
    )
    assert json.loads(encoded)["and"][0]["value"] == "10.0.0.1"


def test_both_and_and_or_survive_encoding() -> None:
    encoded = json.loads(
        encode_asset_filter(
            {
                "and": [{"property": "ipv4", "operator": "eq", "value": "10.0.0.1"}],
                "or": [{"property": "fqdn", "operator": "match", "value": "example"}],
            }
        )
    )
    assert set(encoded) == {"and", "or"}


@pytest.mark.parametrize(
    ("payload", "fragment"),
    [
        ({}, "at least one"),
        ({"and": []}, "at least one"),
        ({"nope": []}, "only 'and' and 'or'"),
        ({"and": [{"operator": "eq", "value": "x"}]}, "property"),
        ({"and": [{"property": "ipv4", "value": "x"}]}, "operator"),
        ({"and": [{"property": "ipv4", "operator": "eq"}]}, "value"),
        ({"and": [{"property": "ipv4", "operator": "eq", "value": []}]}, "at least one"),
        ({"and": [{"property": "ipv4", "operator": "eq", "value": 7}]}, "string"),
        ({"and": [{"property": "ipv4", "operator": "eq", "value": [1]}]}, "string"),
        ({"and": "not-a-list"}, "array"),
    ],
)
def test_malformed_rules_are_rejected(payload: dict, fragment: str) -> None:
    with pytest.raises(TenableError) as excinfo:
        encode_asset_filter(payload)
    assert fragment in excinfo.value.message


def test_asset_must_be_an_object() -> None:
    with pytest.raises(TenableError):
        encode_asset_filter("not-an-object")


def test_rule_count_limit() -> None:
    rule = {"property": "ipv4", "operator": "eq", "value": "10.0.0.1"}
    ok = encode_asset_filter({"and": [rule] * MAX_RULES_PER_TAG})
    assert len(json.loads(ok)["and"]) == MAX_RULES_PER_TAG

    with pytest.raises(TenableError) as excinfo:
        encode_asset_filter({"and": [rule] * (MAX_RULES_PER_TAG + 1)})
    assert str(MAX_RULES_PER_TAG) in excinfo.value.message


def test_rule_count_limit_spans_and_plus_or() -> None:
    """The 40-rule cap is per tag, not per branch."""
    rule = {"property": "ipv4", "operator": "eq", "value": "10.0.0.1"}
    with pytest.raises(TenableError):
        encode_asset_filter({"and": [rule] * 21, "or": [rule] * 20})


def test_values_per_rule_limit() -> None:
    values = [f"10.0.0.{i}" for i in range(MAX_VALUES_PER_RULE + 1)]
    with pytest.raises(TenableError) as excinfo:
        encode_asset_filter(
            {"and": [{"property": "ipv4", "operator": "eq", "value": values}]}
        )
    assert str(MAX_VALUES_PER_RULE) in excinfo.value.message


def test_encoding_is_compact_and_ordered() -> None:
    """Compact separators keep the string byte-stable for Terraform's comparison."""
    encoded = encode_asset_filter(
        {"and": [{"property": "ipv4", "operator": "eq", "value": "10.0.0.1"}]}
    )
    assert ", " not in encoded
    assert encoded == '{"and":[{"field":"ipv4","operator":"eq","value":"10.0.0.1"}]}'
