"""Scanners (read-only).

Scanners are provisioned by Tenable, not by the API, so there is no create or
delete here. Without ``MOCK_SEED`` the list is empty.
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Request

from ..errors import not_found
from ._common import get_store

router = APIRouter(tags=["scanners"])


@router.get("/scanners")
async def list_scanners(request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        return {"scanners": [dict(s) for s in store.scanners.values()]}


@router.get("/scanners/{scanner_id}")
async def get_scanner(scanner_id: int, request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        scanner = store.scanners.get(scanner_id)
        if scanner is None:
            raise not_found(f"Scanner {scanner_id} was not found.")
        return dict(scanner)
