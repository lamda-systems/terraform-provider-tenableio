"""Scan exclusions."""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Request, Response

from ..errors import bad_request, not_found
from ..models import ExclusionCreate, ExclusionSchedule, ExclusionUpdate
from ..store import Store, unix_seconds
from ._common import get_settings, get_store, reject_unknown, resolve_on_omit

router = APIRouter(tags=["exclusions"])

MAX_DESCRIPTION = 3000


def _schedule_payload(schedule: ExclusionSchedule | None) -> dict[str, Any]:
    """Render a schedule, defaulting to the disabled one the API returns.

    A one-time exclusion still carries ``enabled: false`` rather than a null
    schedule, so the shape is always present.
    """
    if schedule is None:
        return {"enabled": False}
    payload: dict[str, Any] = {"enabled": schedule.enabled}
    for key in ("starttime", "endtime", "timezone", "rrules"):
        value = getattr(schedule, key)
        if value is not None:
            payload[key] = value
    return payload


def _validate(name: str, members: str, description: str | None) -> None:
    if not name or not name.strip():
        raise bad_request("The exclusion name is required.")
    if not members or not members.strip():
        raise bad_request("The exclusion members list is required.")
    if description is not None and len(description) > MAX_DESCRIPTION:
        raise bad_request(
            f"The exclusion description exceeds the maximum length of "
            f"{MAX_DESCRIPTION} characters."
        )


def _find(store: Store, exclusion_id: int) -> dict[str, Any]:
    exclusion = store.exclusions.get(exclusion_id)
    if exclusion is None:
        raise not_found(f"Exclusion {exclusion_id} was not found.")
    return exclusion


@router.post("/exclusions")
async def create_exclusion(body: ExclusionCreate, request: Request) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)
    _validate(body.name, body.members, body.description)

    with store.lock:
        if body.network_id and body.network_id not in store.networks:
            raise bad_request(
                f"The specified network {body.network_id} does not exist.",
                code="not_found",
            )
        stamp = unix_seconds(store.now())
        exclusion = {
            "id": store.next_id(),
            "name": body.name,
            "description": body.description or "",
            "members": body.members,
            "network_id": body.network_id or "",
            "schedule": _schedule_payload(body.schedule),
            "creation_date": stamp,
            "last_modification_date": stamp,
        }
        store.exclusions[exclusion["id"]] = exclusion
        return dict(exclusion)


@router.get("/exclusions")
async def list_exclusions(request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        return {"exclusions": [dict(e) for e in store.exclusions.values()]}


@router.get("/exclusions/{exclusion_id}")
async def get_exclusion(exclusion_id: int, request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        return dict(_find(store, exclusion_id))


@router.put("/exclusions/{exclusion_id}")
async def update_exclusion(
    exclusion_id: int, body: ExclusionUpdate, request: Request
) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)
    _validate(body.name, body.members, body.description)

    with store.lock:
        exclusion = _find(store, exclusion_id)
        if body.network_id and body.network_id not in store.networks:
            raise bad_request(
                f"The specified network {body.network_id} does not exist.",
                code="not_found",
            )

        exclusion["name"] = body.name
        exclusion["members"] = body.members
        exclusion["description"] = resolve_on_omit(
            body,
            "description",
            exclusion["description"],
            settings.quirks.on_omitted_description,
        )
        if body.network_id is not None:
            exclusion["network_id"] = body.network_id
        if body.schedule is not None:
            exclusion["schedule"] = _schedule_payload(body.schedule)
        exclusion["last_modification_date"] = unix_seconds(store.now())
        return dict(exclusion)


@router.delete("/exclusions/{exclusion_id}")
async def delete_exclusion(exclusion_id: int, request: Request) -> Response:
    store = get_store(request)
    with store.lock:
        _find(store, exclusion_id)
        del store.exclusions[exclusion_id]
    return Response(status_code=200)
