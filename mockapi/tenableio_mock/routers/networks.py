"""Networks."""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Request, Response

from ..errors import bad_request, not_found
from ..models import NetworkCreate, NetworkUpdate
from ..store import Store, unix_seconds
from ._common import get_settings, get_store, reject_unknown, resolve_on_omit

router = APIRouter(tags=["networks"])

MAX_NETWORK_NAME = 255
MAX_DESCRIPTION = 3000
DEFAULT_ASSETS_TTL_DAYS = 180
MIN_TTL_DAYS = 14
MAX_TTL_DAYS = 365


def _validate(name: str, description: str | None, ttl: int | None) -> None:
    if not name or not name.strip():
        raise bad_request("The network name is required.")
    if len(name) > MAX_NETWORK_NAME:
        raise bad_request(
            f"The network name exceeds the maximum length of {MAX_NETWORK_NAME} characters."
        )
    if description is not None and len(description) > MAX_DESCRIPTION:
        raise bad_request(
            f"The network description exceeds the maximum length of "
            f"{MAX_DESCRIPTION} characters."
        )
    if ttl is not None and not MIN_TTL_DAYS <= ttl <= MAX_TTL_DAYS:
        raise bad_request(
            f"assets_ttl_days must be between {MIN_TTL_DAYS} and {MAX_TTL_DAYS}."
        )


def _find(store: Store, uuid: str) -> dict[str, Any]:
    net = store.networks.get(uuid)
    if net is None:
        raise not_found(f"Network {uuid} was not found.")
    return net


@router.post("/networks")
async def create_network(body: NetworkCreate, request: Request) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)
    _validate(body.name, body.description, body.assets_ttl_days)

    with store.lock:
        if any(n["name"] == body.name for n in store.networks.values()):
            raise bad_request(
                "A network with the name you specified already exists.", code="duplicate"
            )
        stamp = unix_seconds(store.now())
        network = {
            "uuid": store.next_uuid(),
            "name": body.name,
            "description": body.description or "",
            "is_default": False,
            "created_by": settings.user,
            "created_in_seconds": stamp,
            "modified_in_seconds": stamp,
            "scanner_count": 0,
            "assets_ttl_days": (
                body.assets_ttl_days
                if body.assets_ttl_days is not None
                else DEFAULT_ASSETS_TTL_DAYS
            ),
        }
        store.networks[network["uuid"]] = network
        return dict(network)


@router.get("/networks")
async def list_networks(request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        return {"networks": [dict(n) for n in store.networks.values()]}


@router.get("/networks/{uuid}")
async def get_network(uuid: str, request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        return dict(_find(store, uuid))


@router.put("/networks/{uuid}")
async def update_network(
    uuid: str, body: NetworkUpdate, request: Request
) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)
    _validate(body.name, body.description, body.assets_ttl_days)

    with store.lock:
        network = _find(store, uuid)
        clash = next(
            (n for n in store.networks.values() if n["name"] == body.name), None
        )
        if clash is not None and clash["uuid"] != uuid:
            raise bad_request(
                "A network with the name you specified already exists.", code="duplicate"
            )

        network["name"] = body.name
        network["description"] = resolve_on_omit(
            body,
            "description",
            network["description"],
            settings.quirks.on_omitted_description,
        )
        if body.assets_ttl_days is not None:
            network["assets_ttl_days"] = body.assets_ttl_days
        network["modified_in_seconds"] = unix_seconds(store.now())
        return dict(network)


@router.delete("/networks/{uuid}")
async def delete_network(uuid: str, request: Request) -> Response:
    store = get_store(request)
    with store.lock:
        network = _find(store, uuid)
        if network["is_default"]:
            raise bad_request("The default network cannot be deleted.")
        del store.networks[uuid]
    return Response(status_code=200)
