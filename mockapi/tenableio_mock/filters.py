"""Translation between the two shapes Tenable.io uses for dynamic tag rules.

This module is the reason the mock exists in the first place. ``POST`` and
``PUT`` on ``/tags/values`` accept a ``filters`` **object**; every read path
echoes ``filters.asset`` back as a **JSON-formatted string**, and the rules
inside it are spelled differently::

    request   {"asset": {"and": [{"property": "operating_system",
                                  "operator": "equals",
                                  "value": ["FreeBSD"]}]}}

    response  {"asset": "{\\"and\\": [{\\"field\\": \\"operating_system\\",
                                      \\"operator\\": \\"eq\\",
                                      \\"value\\": \\"FreeBSD\\"}]}"}

Three things change at once: the container goes from object to string, the
attribute key goes from ``property`` to ``field``, and the operator goes from a
readable word to a short code. A single-element value list also collapses to a
bare string. A mock that skipped any of these would let a provider that cannot
parse the real response pass its tests.
"""

from __future__ import annotations

import json
from typing import Any

from .errors import bad_request

#: Operators the API accepts, as reported by ``GET /tags/assets/filters``.
#: This set is closed and documented, so an operator outside it is rejected.
SHORT_OPERATORS: frozenset[str] = frozenset(
    {
        "eq",
        "neq",
        "match",
        "nmatch",
        "wc",
        "nwc",
        "exists",
        "nexists",
        "date-lt",
        "date-gt",
        "set-has",
        "set-hasnot",
        "set-hasonly",
        "eq-hide",
    }
)

#: Readable spellings accepted on the request side, mapped to the short code
#: the response echoes. Requests may use either form; responses always use the
#: short code, which is what live Tenable.io does.
OPERATOR_ALIASES: dict[str, str] = {
    "equals": "eq",
    "is equal to": "eq",
    "not equals": "neq",
    "does not equal": "neq",
    "is not equal to": "neq",
    "contains": "match",
    "does not contain": "nmatch",
    "wildcard": "wc",
    "not wildcard": "nwc",
    "does not match wildcard": "nwc",
    "does not exist": "nexists",
    "date less than": "date-lt",
    "date greater than": "date-gt",
    "has": "set-has",
    "does not have": "set-hasnot",
    "has only": "set-hasonly",
}

MAX_RULES_PER_TAG = 40
MAX_VALUES_PER_RULE = 1024


def normalise_operator(operator: str) -> str:
    """Map a request-side operator to the short code used in responses.

    Accepts either spelling. Unknown operators are rejected rather than passed
    through: the operator vocabulary is a closed, documented set, so a value
    outside it is a caller mistake that production would also reject.

    Note the asymmetry with ``property``, which is *not* validated. The set of
    taggable asset attributes depends on which connectors a tenant has
    configured, so it is open-ended and the mock has no basis to reject one.
    """
    raw = (operator or "").strip()
    if raw in SHORT_OPERATORS:
        return raw
    aliased = OPERATOR_ALIASES.get(raw.lower())
    if aliased is not None:
        return aliased
    known = ", ".join(sorted(SHORT_OPERATORS))
    raise bad_request(
        f"Invalid filter operator {operator!r}. Expected one of: {known}."
    )


def _rule_to_response(rule: dict[str, Any]) -> dict[str, Any]:
    """Convert one request-shaped rule to its response-shaped equivalent."""
    if not isinstance(rule, dict):
        raise bad_request("Each asset filter rule must be an object.")

    # The request side names the attribute "property"; tolerate "field" too,
    # since that is what a caller round-tripping a previous response will send.
    field = rule.get("property") or rule.get("field")
    if not isinstance(field, str) or not field.strip():
        raise bad_request("Each asset filter rule requires a non-empty 'property'.")

    operator = rule.get("operator")
    if not isinstance(operator, str):
        raise bad_request(f"Rule for {field!r} requires an 'operator'.")

    value = rule.get("value")
    if value is None:
        raise bad_request(f"Rule for {field!r} requires a 'value'.")

    # The API accepts a bare string or an array of strings.
    if isinstance(value, str):
        values = [value]
    elif isinstance(value, list):
        if not all(isinstance(v, str) for v in value):
            raise bad_request(f"Rule for {field!r} must have string values.")
        values = list(value)
    else:
        raise bad_request(
            f"Rule for {field!r} must have a string or array-of-strings 'value'."
        )

    if not values:
        raise bad_request(f"Rule for {field!r} requires at least one value.")
    if len(values) > MAX_VALUES_PER_RULE:
        raise bad_request(
            f"Rule for {field!r} has {len(values)} values; "
            f"the maximum is {MAX_VALUES_PER_RULE}."
        )

    return {
        "field": field,
        "operator": normalise_operator(operator),
        # Collapse a single value back to a bare string, mirroring the
        # documented response example.
        "value": values[0] if len(values) == 1 else values,
    }


def encode_asset_filter(asset: Any) -> str:
    """Serialise a request-side ``filters.asset`` object into the response string.

    Returns the JSON-formatted string that every read path echoes back. Raises
    :class:`~tenableio_mock.errors.TenableError` for anything the real API
    would reject.
    """
    if not isinstance(asset, dict):
        raise bad_request("'filters.asset' must be an object.")

    unknown = set(asset) - {"and", "or"}
    if unknown:
        raise bad_request(
            f"'filters.asset' accepts only 'and' and 'or' (got {sorted(unknown)!r})."
        )

    out: dict[str, list[dict[str, Any]]] = {}
    total = 0
    for key in ("and", "or"):
        rules = asset.get(key)
        if rules is None:
            continue
        if not isinstance(rules, list):
            raise bad_request(f"'filters.asset.{key}' must be an array of rules.")
        if not rules:
            continue
        out[key] = [_rule_to_response(r) for r in rules]
        total += len(rules)

    if total == 0:
        raise bad_request("'filters.asset' requires at least one 'and' or 'or' rule.")
    if total > MAX_RULES_PER_TAG:
        raise bad_request(
            f"Tag has {total} rules; the maximum is {MAX_RULES_PER_TAG}."
        )

    # separators without spaces keeps the encoded string compact and, more
    # importantly, byte-stable across reads -- Terraform compares state
    # verbatim, so a re-encoding that shifted whitespace would look like drift.
    return json.dumps(out, separators=(",", ":"), sort_keys=False)


#: Filter definitions returned by ``GET /tags/assets/filters``. Trimmed to a
#: representative slice of what a real container reports: one entry per control
#: type so a consumer sees an ``entry`` with a regex, a ``dropdown`` with
#: ``{name, value}`` options, and a ``dropdown_multi``.
ASSET_TAG_FILTERS: list[dict[str, Any]] = [
    {
        "name": "asset_class",
        "readable_name": "Asset Class",
        "operators": ["eq", "neq"],
        "control": {
            "type": "dropdown",
            "list": [
                {"name": "Device", "value": "DEVICE"},
                {"name": "Web Application", "value": "WEB_APPLICATION"},
                {"name": "Cloud Resource", "value": "CLOUD_RESOURCE"},
                {"name": "Identity", "value": "IDENTITY"},
            ],
        },
    },
    {
        "name": "ipv4",
        "readable_name": "IPv4 Address",
        "operators": ["eq", "neq", "match", "nmatch"],
        "control": {
            "type": "entry",
            "regex": r"^((\d{1,3}\.){3}\d{1,3})(\/\d{1,2})?$",
            "readable_regex": "e.g. 192.168.0.1 or 192.168.0.0/24",
        },
    },
    {
        "name": "operating_system",
        "readable_name": "Operating System",
        "operators": ["eq", "neq", "match", "nmatch", "wc", "nwc"],
        "control": {
            "type": "entry",
            "regex": ".*",
            "readable_regex": "e.g. Microsoft Windows Server 2016",
        },
    },
    {
        "name": "fqdn",
        "readable_name": "FQDN",
        "operators": ["eq", "neq", "match", "nmatch", "wc", "nwc"],
        "control": {
            "type": "entry",
            "regex": ".*",
            "readable_regex": "e.g. host.example.com",
        },
    },
    {
        "name": "netbios_name",
        "readable_name": "NetBIOS Name",
        "operators": ["eq", "neq", "match", "nmatch"],
        "control": {
            "type": "entry",
            "regex": ".*",
            "readable_regex": "e.g. WORKSTATION01",
        },
    },
    {
        "name": "tenable_uuid",
        "readable_name": "Tenable UUID",
        "operators": ["eq", "neq"],
        "control": {
            "type": "entry",
            "regex": ".*",
            "readable_regex": "e.g. 123e4567e89b12d3a456426655440000",
        },
    },
    {
        "name": "aws_ec2_instance_id",
        "readable_name": "AWS EC2 Instance ID",
        "operators": ["eq", "neq", "match", "nmatch"],
        "control": {
            "type": "entry",
            "regex": ".*",
            "readable_regex": "e.g. i-0abcd1234efgh5678",
        },
    },
    {
        "name": "sources",
        "readable_name": "Source",
        "operators": ["set-has", "set-hasnot", "set-hasonly"],
        "control": {
            "type": "dropdown_multi",
            "list": [
                {"name": "Nessus Scan", "value": "NESSUS_SCAN"},
                {"name": "Nessus Agent", "value": "NESSUS_AGENT"},
                {"name": "AWS", "value": "AWS"},
                {"name": "Azure", "value": "AZURE"},
            ],
        },
    },
    {
        "name": "last_seen",
        "readable_name": "Last Seen",
        "operators": ["date-lt", "date-gt"],
        "control": {
            "type": "entry",
            "regex": r"^\d{4}-\d{2}-\d{2}$",
            "readable_regex": "e.g. 2026-01-31",
        },
    },
]
