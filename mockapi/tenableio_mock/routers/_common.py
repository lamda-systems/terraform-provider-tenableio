"""Helpers shared by the routers."""

from __future__ import annotations

from typing import Any

from fastapi import Request
from pydantic import BaseModel

from ..config import OnOmit, Settings
from ..errors import bad_request
from ..models import unknown_fields, was_provided
from ..store import Store


def get_store(request: Request) -> Store:
    return request.app.state.store


def get_settings(request: Request) -> Settings:
    return request.app.state.settings


def reject_unknown(model: BaseModel, settings: Settings) -> None:
    """Enforce the ``reject_unknown_fields`` lint quirk, when enabled."""
    if not settings.quirks.reject_unknown_fields:
        return
    extras = unknown_fields(model)
    if extras:
        raise bad_request(
            f"Unrecognized field(s) in request body: {', '.join(extras)}."
        )


def resolve_on_omit(
    model: BaseModel, name: str, current: Any, policy: OnOmit, absent: Any = ""
) -> Any:
    """Decide an update's new value for a key that may be absent from the body.

    Present in the body wins outright, including when it is empty -- an
    explicit ``""`` is a request to clear. Absent falls to *policy*: either keep
    what is stored, or reset to *absent*.
    """
    if was_provided(model, name):
        value = getattr(model, name)
        return absent if value is None else value
    if policy is OnOmit.PRESERVES:
        return current
    return absent


def require_len(value: str | None, limit: int, label: str) -> None:
    if value is not None and len(value) > limit:
        raise bad_request(f"{label} exceeds the maximum length of {limit} characters.")


def access_control() -> dict[str, Any]:
    """The permissions block every tag endpoint returns.

    The provider ignores it, but omitting it entirely would make the mock's
    responses structurally narrower than production's, which is the kind of gap
    that hides a deserialisation bug.
    """
    return {
        "current_user_permissions": ["ALL", "CAN_EDIT", "CAN_SET_PERMISSIONS"],
        "defined_domain_permissions": ["ALL", "CAN_EDIT", "CAN_SET_PERMISSIONS"],
        "all_users_permissions": ["CAN_EDIT"],
        "current_domain_permissions": [],
        "version": 0,
    }


def pagination(total: int, limit: int = 5000, sort_by: str = "name") -> dict[str, Any]:
    return {
        "offset": 0,
        "limit": limit,
        "total": total,
        "sort": [{"name": sort_by, "order": "asc"}],
    }
