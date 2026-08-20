"""Scans.

The read and write shapes diverge more here than anywhere else outside tags,
and the divergence is faithful:

* ``POST /scans`` returns the new scan wrapped as ``{"scan": {...}}``.
* ``GET /scans/{id}`` returns ``{"info": {...}}`` -- a *different* object whose
  keys are renamed: the identifier is ``object_id`` rather than ``id``, the
  target list is ``targets`` rather than ``text_targets``, the recipient list is
  ``notification_email_address`` rather than ``emails``, and the template is
  reported under ``scanner_name``.
* ``GET /scans`` returns a slimmer list item again.
* ``PUT /scans/{id}`` returns an empty body.

Anything that flattens these into one shape is not reproducing the API.
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Request, Response

from ..errors import bad_request, not_found
from ..models import ScanCreate, ScanSettings, ScanUpdate
from ..store import Store, unix_seconds
from ._common import get_settings, get_store, reject_unknown, resolve_on_omit

router = APIRouter(tags=["scans"])

MAX_DESCRIPTION = 3000
LAUNCH_VALUES = {"ON_DEMAND", "DAILY", "WEEKLY", "MONTHLY", "YEARLY"}


def _validate(settings_body: ScanSettings, store: Store) -> None:
    if not settings_body.name or not settings_body.name.strip():
        raise bad_request("The scan name is required.")
    if settings_body.description is not None and len(settings_body.description) > MAX_DESCRIPTION:
        raise bad_request(
            f"The scan description exceeds the maximum length of "
            f"{MAX_DESCRIPTION} characters."
        )
    if settings_body.launch is not None and settings_body.launch not in LAUNCH_VALUES:
        raise bad_request(
            f"launch must be one of: {', '.join(sorted(LAUNCH_VALUES))}."
        )
    if settings_body.folder_id and settings_body.folder_id not in store.folders:
        raise bad_request(
            f"The specified folder {settings_body.folder_id} does not exist.",
            code="not_found",
        )
    if settings_body.scanner_id and settings_body.scanner_id not in store.scanners:
        raise bad_request(
            f"The specified scanner {settings_body.scanner_id} does not exist.",
            code="not_found",
        )
    if settings_body.policy_id and settings_body.policy_id not in store.policies:
        raise bad_request(
            f"The specified policy {settings_body.policy_id} does not exist.",
            code="not_found",
        )


def _find(store: Store, scan_id: int) -> dict[str, Any]:
    scan = store.scans.get(scan_id)
    if scan is None:
        raise not_found(f"Scan {scan_id} was not found.")
    return scan


def _detail_payload(scan: dict[str, Any]) -> dict[str, Any]:
    """The shape ``POST /scans`` wraps in ``{"scan": ...}``."""
    return {
        "id": scan["id"],
        "uuid": scan["uuid"],
        "name": scan["name"],
        "description": scan["description"],
        "policy_id": scan["policy_id"],
        "folder_id": scan["folder_id"],
        "scanner_id": scan["scanner_id"],
        "text_targets": scan["text_targets"],
        "starttime": scan["starttime"],
        "rrules": scan["rrules"],
        "timezone": scan["timezone"],
        "emails": scan["emails"],
        "enabled": scan["enabled"],
        "launch": scan["launch"],
        "scan_time_window": scan["scan_time_window"],
        "status": scan["status"],
        "creation_date": scan["creation_date"],
        "last_modification_date": scan["last_modification_date"],
        "type": scan["type"],
    }


def _info_payload(scan: dict[str, Any]) -> dict[str, Any]:
    """The renamed shape ``GET /scans/{id}`` wraps in ``{"info": ...}``."""
    return {
        "object_id": scan["id"],
        "uuid": scan["uuid"],
        "name": scan["name"],
        "description": scan["description"],
        "policy_id": scan["policy_id"],
        "folder_id": scan["folder_id"],
        "scanner_id": scan["scanner_id"],
        "targets": scan["text_targets"],
        "starttime": scan["starttime"],
        "rrules": scan["rrules"],
        "timezone": scan["timezone"],
        "notification_email_address": scan["emails"],
        "enabled": scan["enabled"],
        "launch": scan["launch"],
        "scan_time_window": scan["scan_time_window"],
        "status": scan["status"],
        "creation_date": scan["creation_date"],
        "last_modification_date": scan["last_modification_date"],
        "scan_type": scan["type"],
        "scanner_name": scan["template_uuid"],
    }


def _list_payload(scan: dict[str, Any]) -> dict[str, Any]:
    return {
        "id": scan["id"],
        "uuid": scan["uuid"],
        "name": scan["name"],
        "description": scan["description"],
        "folder_id": scan["folder_id"],
        "type": scan["type"],
        "status": scan["status"],
        "enabled": scan["enabled"],
        "creation_date": scan["creation_date"],
        "last_modification_date": scan["last_modification_date"],
    }


@router.post("/scans")
async def create_scan(body: ScanCreate, request: Request) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)
    if not body.uuid or not body.uuid.strip():
        raise bad_request("The template uuid is required.")

    with store.lock:
        _validate(body.settings, store)
        stamp = unix_seconds(store.now())
        s = body.settings
        scan = {
            "id": store.next_id(),
            "uuid": store.next_uuid(),
            "name": s.name,
            "description": s.description or "",
            "policy_id": s.policy_id or 0,
            "folder_id": s.folder_id or 0,
            "scanner_id": s.scanner_id or 0,
            "text_targets": s.text_targets or "",
            "tag_targets": s.tag_targets or [],
            "starttime": s.starttime or "",
            "rrules": s.rrules or "",
            "timezone": s.timezone or "",
            "emails": s.emails or "",
            "enabled": s.enabled,
            "launch": s.launch or "ON_DEMAND",
            "scan_time_window": s.scan_time_window or 0,
            "status": "empty",
            "creation_date": stamp,
            "last_modification_date": stamp,
            "type": "public",
            "template_uuid": body.uuid,
        }
        store.scans[scan["id"]] = scan
        return {"scan": _detail_payload(scan)}


@router.get("/scans")
async def list_scans(request: Request, folder_id: int | None = None) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        scans = list(store.scans.values())
        if folder_id is not None:
            scans = [s for s in scans if s["folder_id"] == folder_id]
        return {"scans": [_list_payload(s) for s in scans]}


@router.get("/scans/{scan_id}")
async def get_scan(scan_id: int, request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        return {"info": _info_payload(_find(store, scan_id))}


@router.put("/scans/{scan_id}")
async def update_scan(scan_id: int, body: ScanUpdate, request: Request) -> Response:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)

    with store.lock:
        scan = _find(store, scan_id)
        _validate(body.settings, store)
        s = body.settings

        scan["name"] = s.name
        scan["description"] = resolve_on_omit(
            s, "description", scan["description"], settings.quirks.on_omitted_description
        )
        scan["enabled"] = s.enabled
        for field, key in (
            ("policy_id", "policy_id"),
            ("folder_id", "folder_id"),
            ("scanner_id", "scanner_id"),
            ("text_targets", "text_targets"),
            ("tag_targets", "tag_targets"),
            ("starttime", "starttime"),
            ("rrules", "rrules"),
            ("timezone", "timezone"),
            ("emails", "emails"),
            ("launch", "launch"),
            ("scan_time_window", "scan_time_window"),
        ):
            value = getattr(s, field)
            if value is not None:
                scan[key] = value
        if body.uuid:
            scan["template_uuid"] = body.uuid
        scan["last_modification_date"] = unix_seconds(store.now())
    return Response(status_code=200)


@router.delete("/scans/{scan_id}")
async def delete_scan(scan_id: int, request: Request) -> Response:
    store = get_store(request)
    with store.lock:
        _find(store, scan_id)
        del store.scans[scan_id]
    return Response(status_code=200)
