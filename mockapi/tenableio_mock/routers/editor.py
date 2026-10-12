"""Tenable-provided scan templates (read-only).

``GET /editor/{type}/templates`` lists them; ``GET
/editor/{type}/templates/{uuid}`` describes one. A uuid is only found in the
list it came from -- see ``templates`` for why the lists do not share uuids.
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter

from ..errors import bad_request, not_found
from ..templates import TEMPLATE_TYPES, find_template, list_templates, template_details

router = APIRouter(tags=["editor"])


def _check_type(template_type: str) -> None:
    if template_type not in TEMPLATE_TYPES:
        raise bad_request(
            f"Invalid template type {template_type!r}; "
            f"expected one of: {', '.join(TEMPLATE_TYPES)}."
        )


@router.get("/editor/{template_type}/templates")
async def list_editor_templates(template_type: str) -> dict[str, Any]:
    _check_type(template_type)
    return {"templates": list_templates(template_type)}


@router.get("/editor/{template_type}/templates/{template_uuid}")
async def get_editor_template(template_type: str, template_uuid: str) -> dict[str, Any]:
    _check_type(template_type)
    entry = find_template(template_type, template_uuid)
    if entry is None:
        raise not_found(f"Template {template_uuid} was not found.")
    return template_details(entry)
