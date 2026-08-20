"""Request models.

Two conventions run through this module.

``extra="allow"`` — real Tenable.io ignores fields it does not recognise, so
the models do too. The ``reject_unknown_fields`` quirk turns that into a 400 by
inspecting ``model_extra``; it is a lint mode, not the default.

Optional fields default to ``None`` and are read through
:func:`was_provided` rather than by truthiness. The difference between "the key
was absent" and "the key was present and empty" decides whether an update
clears a field or leaves it alone, and it is precisely the distinction a
provider using ``omitempty`` erases on the wire.
"""

from __future__ import annotations

from typing import Any

from pydantic import BaseModel, ConfigDict


class _Body(BaseModel):
    model_config = ConfigDict(extra="allow")


def was_provided(model: BaseModel, name: str) -> bool:
    """Report whether ``name`` was physically present in the request body."""
    return name in model.model_fields_set


def unknown_fields(model: BaseModel) -> list[str]:
    return sorted(model.model_extra or {})


# -- tags ------------------------------------------------------------------


class TagCategoryCreate(_Body):
    name: str
    description: str | None = None


class TagCategoryUpdate(_Body):
    # Tenable.io documents name as required on PUT: the endpoint replaces the
    # category rather than patching it.
    name: str
    description: str | None = None


class TagValueCreate(_Body):
    # Exactly one of category_uuid / category_name must be supplied; the check
    # lives in the router so it can raise a Tenable-shaped error.
    category_uuid: str | None = None
    category_name: str | None = None
    category_description: str | None = None
    value: str
    description: str | None = None
    filters: dict[str, Any] | None = None


class TagValueUpdate(_Body):
    value: str | None = None
    description: str | None = None
    filters: dict[str, Any] | None = None


# -- folders ---------------------------------------------------------------


class FolderCreate(_Body):
    name: str


class FolderEdit(_Body):
    name: str


# -- networks --------------------------------------------------------------


class NetworkCreate(_Body):
    name: str
    description: str | None = None
    assets_ttl_days: int | None = None


class NetworkUpdate(_Body):
    name: str
    description: str | None = None
    assets_ttl_days: int | None = None


# -- exclusions ------------------------------------------------------------


class ExclusionSchedule(_Body):
    enabled: bool = False
    starttime: str | None = None
    endtime: str | None = None
    timezone: str | None = None
    rrules: str | None = None


class ExclusionCreate(_Body):
    name: str
    description: str | None = None
    members: str
    network_id: str | None = None
    schedule: ExclusionSchedule | None = None


class ExclusionUpdate(_Body):
    name: str
    description: str | None = None
    members: str
    network_id: str | None = None
    schedule: ExclusionSchedule | None = None


# -- agent groups ----------------------------------------------------------


class AgentGroupCreate(_Body):
    name: str


# -- policies --------------------------------------------------------------


class PolicySettings(_Body):
    name: str
    description: str | None = None
    visibility: str | None = None


class PolicyCreate(_Body):
    uuid: str
    settings: PolicySettings


class PolicyUpdate(_Body):
    uuid: str | None = None
    settings: PolicySettings


# -- scans -----------------------------------------------------------------


class ScanSettings(_Body):
    name: str
    description: str | None = None
    policy_id: int | None = None
    folder_id: int | None = None
    scanner_id: int | None = None
    text_targets: str | None = None
    tag_targets: list[str] | None = None
    file_targets: str | None = None
    launch: str | None = None
    enabled: bool = False
    starttime: str | None = None
    rrules: str | None = None
    timezone: str | None = None
    emails: str | None = None
    scan_time_window: int | None = None


class ScanCreate(_Body):
    uuid: str
    settings: ScanSettings


class ScanUpdate(_Body):
    uuid: str | None = None
    settings: ScanSettings
