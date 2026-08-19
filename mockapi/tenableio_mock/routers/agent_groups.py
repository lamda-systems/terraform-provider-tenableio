"""Agent groups.

These live under ``/scanners/null/agent-groups``. The literal ``null`` in the
path is Tenable.io's own spelling for "the container's cloud scanner", not a
placeholder the caller is meant to substitute.
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Request, Response

from ..errors import bad_request, not_found
from ..models import AgentGroupCreate
from ..store import unix_seconds
from ._common import get_settings, get_store, reject_unknown

router = APIRouter(tags=["agent-groups"])

PREFIX = "/scanners/null/agent-groups"


@router.post(PREFIX)
async def create_agent_group(body: AgentGroupCreate, request: Request) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)
    if not body.name or not body.name.strip():
        raise bad_request("The agent group name is required.")

    with store.lock:
        if any(g["name"] == body.name for g in store.agent_groups.values()):
            raise bad_request(
                "An agent group with the name you specified already exists.",
                code="duplicate",
            )
        stamp = unix_seconds(store.now())
        group = {
            "id": store.next_id(),
            "name": body.name,
            "owner_id": 1,
            "owner": settings.user,
            "shared": 1,
            "agents_count": 0,
            "creation_date": stamp,
            "timestamp": stamp,
        }
        store.agent_groups[group["id"]] = group
        return dict(group)


@router.get(PREFIX)
async def list_agent_groups(request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        return {"groups": [dict(g) for g in store.agent_groups.values()]}


@router.get(PREFIX + "/{group_id}")
async def get_agent_group(group_id: int, request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        group = store.agent_groups.get(group_id)
        if group is None:
            raise not_found(f"Agent group {group_id} was not found.")
        return dict(group)


@router.delete(PREFIX + "/{group_id}")
async def delete_agent_group(group_id: int, request: Request) -> Response:
    store = get_store(request)
    with store.lock:
        if group_id not in store.agent_groups:
            raise not_found(f"Agent group {group_id} was not found.")
        del store.agent_groups[group_id]
    return Response(status_code=200)
