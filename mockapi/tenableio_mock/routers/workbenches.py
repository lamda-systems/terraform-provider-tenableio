"""Workbench assets (read-only).

Assets arrive from scans and connectors, so nothing creates them through the
API. The list endpoint accepts the ``date_range`` and indexed ``filter.N.*``
query parameters the provider builds; filtering is applied only for the small
number of fields a data source is likely to select on, which is enough to prove
the query string is assembled correctly.
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Request

from ..errors import not_found
from ._common import get_store

router = APIRouter(tags=["workbenches"])

#: Asset fields the mock knows how to filter on. Anything else is accepted and
#: ignored, matching an API that silently drops filters it does not recognise.
FILTERABLE = {"ipv4", "fqdn", "operating_system", "netbios_name", "hostname"}


def _list_payload(asset: dict[str, Any]) -> dict[str, Any]:
    return {
        "id": asset["id"],
        "has_agent": asset["has_agent"],
        "has_plugin_results": asset["has_plugin_results"],
        "fqdn": asset["fqdn"],
        "ipv4": asset["ipv4"],
        "ipv6": asset["ipv6"],
        "mac_address": asset["mac_address"],
        "netbios_name": asset["netbios_name"],
        "operating_system": asset["operating_system"],
        "agent_name": asset["agent_name"],
        "last_seen": asset["last_seen"],
        "first_seen": asset["first_seen"],
    }


def _parse_filters(params: dict[str, str]) -> list[tuple[str, str, str]]:
    """Reassemble the indexed ``filter.N.filter|quality|value`` triples."""
    indexed: dict[str, dict[str, str]] = {}
    for key, value in params.items():
        parts = key.split(".")
        if len(parts) != 3 or parts[0] != "filter":
            continue
        indexed.setdefault(parts[1], {})[parts[2]] = value
    out = []
    for group in indexed.values():
        field = group.get("filter", "")
        if field:
            out.append((field, group.get("quality", "eq"), group.get("value", "")))
    return out


def _matches(asset: dict[str, Any], field: str, quality: str, value: str) -> bool:
    if field not in FILTERABLE:
        return True
    haystack = asset.get(field) or []
    if quality in ("eq", "equal"):
        return value in haystack
    if quality in ("neq", "nequal"):
        return value not in haystack
    if quality in ("match", "wc"):
        return any(value.lower() in str(item).lower() for item in haystack)
    if quality == "nmatch":
        return not any(value.lower() in str(item).lower() for item in haystack)
    return True


@router.get("/workbenches/assets")
async def list_assets(request: Request) -> dict[str, Any]:
    store = get_store(request)
    params = dict(request.query_params)
    filters = _parse_filters(params)

    with store.lock:
        assets = list(store.assets.values())

    for field, quality, value in filters:
        assets = [a for a in assets if _matches(a, field, quality, value)]

    return {
        "assets": [_list_payload(a) for a in assets],
        "total": len(assets),
    }


@router.get("/workbenches/assets/{asset_id}/info")
async def get_asset(asset_id: str, request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        asset = store.assets.get(asset_id)
        if asset is None:
            raise not_found(f"Asset {asset_id} was not found.")
        return {"info": dict(asset)}
