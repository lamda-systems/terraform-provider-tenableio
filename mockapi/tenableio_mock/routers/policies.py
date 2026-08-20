"""Scan policies.

Three different shapes are in play and they do not match each other, which is
faithful to production:

* ``POST /policies`` takes ``{uuid, settings{...}}`` and returns only
  ``{policy_id, policy_name}``.
* ``GET /policies/{id}`` returns a flat detail object.
* ``GET /policies`` returns a list of those detail objects.
* ``PUT /policies/{id}`` returns an empty body.

The ``uuid`` on the request is the *template* the policy derives from, not the
policy's own identifier.
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Request, Response

from ..errors import bad_request, not_found
from ..models import PolicyCreate, PolicyUpdate
from ..store import Store, unix_seconds
from ._common import get_settings, get_store, reject_unknown, resolve_on_omit

router = APIRouter(tags=["policies"])

MAX_DESCRIPTION = 3000
VISIBILITIES = {"private", "shared"}


def _validate(name: str, description: str | None, visibility: str | None) -> None:
    if not name or not name.strip():
        raise bad_request("The policy name is required.")
    if description is not None and len(description) > MAX_DESCRIPTION:
        raise bad_request(
            f"The policy description exceeds the maximum length of "
            f"{MAX_DESCRIPTION} characters."
        )
    if visibility is not None and visibility not in VISIBILITIES:
        raise bad_request(
            f"visibility must be one of: {', '.join(sorted(VISIBILITIES))}."
        )


def _find(store: Store, policy_id: int) -> dict[str, Any]:
    policy = store.policies.get(policy_id)
    if policy is None:
        raise not_found(f"Policy {policy_id} was not found.")
    return policy


@router.post("/policies")
async def create_policy(body: PolicyCreate, request: Request) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)
    _validate(body.settings.name, body.settings.description, body.settings.visibility)
    if not body.uuid or not body.uuid.strip():
        raise bad_request("The template uuid is required.")

    with store.lock:
        stamp = unix_seconds(store.now())
        policy = {
            "id": store.next_id(),
            # The policy gets its own uuid; body.uuid identified the template.
            "uuid": store.next_uuid(),
            "name": body.settings.name,
            "description": body.settings.description or "",
            "owner": settings.user,
            "owner_id": 1,
            "visibility": body.settings.visibility or "private",
            "creation_date": stamp,
            "last_modification_date": stamp,
            "no_target": "false",
            "template_uuid": body.uuid,
        }
        store.policies[policy["id"]] = policy
        return {"policy_id": policy["id"], "policy_name": policy["name"]}


@router.get("/policies")
async def list_policies(request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        return {"policies": [dict(p) for p in store.policies.values()]}


@router.get("/policies/{policy_id}")
async def get_policy(policy_id: int, request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        return dict(_find(store, policy_id))


@router.put("/policies/{policy_id}")
async def update_policy(
    policy_id: int, body: PolicyUpdate, request: Request
) -> Response:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)
    _validate(body.settings.name, body.settings.description, body.settings.visibility)

    with store.lock:
        policy = _find(store, policy_id)
        policy["name"] = body.settings.name
        policy["description"] = resolve_on_omit(
            body.settings,
            "description",
            policy["description"],
            settings.quirks.on_omitted_description,
        )
        if body.settings.visibility is not None:
            policy["visibility"] = body.settings.visibility
        if body.uuid:
            policy["template_uuid"] = body.uuid
        policy["last_modification_date"] = unix_seconds(store.now())
    return Response(status_code=200)


@router.delete("/policies/{policy_id}")
async def delete_policy(policy_id: int, request: Request) -> Response:
    store = get_store(request)
    with store.lock:
        _find(store, policy_id)
        del store.policies[policy_id]
    return Response(status_code=200)
