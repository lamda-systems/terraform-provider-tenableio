"""Folders.

Note the response shapes: ``POST /folders`` returns only the new ``id``, and
both ``PUT`` and ``DELETE`` return an empty body. Callers that need the whole
folder have to go back to ``GET /folders`` -- there is no per-folder GET, which
is why the provider's client filters the list.
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Request, Response

from ..errors import bad_request, not_found
from ..models import FolderCreate, FolderEdit
from ._common import get_settings, get_store, reject_unknown

router = APIRouter(tags=["folders"])

MAX_FOLDER_NAME = 255


def _validate_name(name: str) -> None:
    if not name or not name.strip():
        raise bad_request("The folder name is required.")
    if len(name) > MAX_FOLDER_NAME:
        raise bad_request(
            f"The folder name exceeds the maximum length of {MAX_FOLDER_NAME} characters."
        )


@router.post("/folders")
async def create_folder(body: FolderCreate, request: Request) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)
    _validate_name(body.name)

    with store.lock:
        folder = {
            "id": store.next_id(),
            "name": body.name,
            "type": "custom",
            "custom": 1,
            "unread_count": 0,
            "default_tag": 0,
        }
        store.folders[folder["id"]] = folder
        # Only the identifier comes back on create.
        return {"id": folder["id"]}


@router.get("/folders")
async def list_folders(request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        return {"folders": [dict(f) for f in store.folders.values()]}


@router.put("/folders/{folder_id}")
async def edit_folder(folder_id: int, body: FolderEdit, request: Request) -> Response:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)
    _validate_name(body.name)

    with store.lock:
        folder = store.folders.get(folder_id)
        if folder is None:
            raise not_found(f"Folder {folder_id} was not found.")
        if folder["custom"] == 0:
            raise bad_request("A system folder cannot be renamed.")
        folder["name"] = body.name
    return Response(status_code=200)


@router.delete("/folders/{folder_id}")
async def delete_folder(folder_id: int, request: Request) -> Response:
    store = get_store(request)
    with store.lock:
        folder = store.folders.get(folder_id)
        if folder is None:
            raise not_found(f"Folder {folder_id} was not found.")
        if folder["custom"] == 0:
            raise bad_request("A system folder cannot be deleted.")
        del store.folders[folder_id]
    return Response(status_code=200)
