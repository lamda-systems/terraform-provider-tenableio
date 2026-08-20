"""Tag categories, tag values, and the asset filter catalogue.

The behaviours worth knowing about, all of them deliberate:

* ``POST /tags/categories`` rejects a duplicate name with 400. It does **not**
  return the existing category. A mock that quietly upserts here will let a
  provider ship a bug where Terraform's planned ``name`` and the applied
  ``name`` disagree, which Terraform reports as "Provider produced inconsistent
  result after apply".
* ``POST /tags/values`` *does* create a category on demand when given
  ``category_name``. That asymmetry with the endpoint above is real.
* Every read path returns ``filters.asset`` as a JSON-formatted string with
  ``field`` keys and short operator codes, never as the object the write path
  accepted. See :mod:`tenableio_mock.filters`.
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Request

from ..config import Settings
from ..errors import bad_request, not_found
from ..filters import ASSET_TAG_FILTERS, encode_asset_filter
from ..models import (
    TagCategoryCreate,
    TagCategoryUpdate,
    TagValueCreate,
    TagValueUpdate,
    was_provided,
)
from ..store import Store, rfc3339_millis
from ._common import (
    access_control,
    get_settings,
    get_store,
    pagination,
    reject_unknown,
    require_len,
    resolve_on_omit,
)

router = APIRouter(tags=["tags"])

MAX_CATEGORY_NAME = 127
MAX_DESCRIPTION = 3000
MAX_CATEGORIES = 100
MAX_VALUES_PER_CATEGORY = 100_000


# -- category helpers ------------------------------------------------------


def _normalise_category_name(name: str, settings: Settings) -> str:
    return name.lower() if settings.quirks.lowercase_category_names else name


def _validate_category_name(name: str) -> None:
    if not name or not name.strip():
        raise bad_request("The category name is required.")
    if len(name) > MAX_CATEGORY_NAME:
        raise bad_request(
            f"The category name exceeds the maximum length of {MAX_CATEGORY_NAME} characters."
        )
    if ":" in name:
        raise bad_request("The category name cannot contain a colon.")


def _category_payload(category: dict[str, Any]) -> dict[str, Any]:
    return dict(category)


def _find_category(store: Store, uuid: str) -> dict[str, Any]:
    category = store.categories.get(uuid)
    if category is None:
        raise not_found(f"Tag category {uuid} was not found.")
    return category


# -- categories ------------------------------------------------------------


@router.post("/tags/categories")
async def create_tag_category(body: TagCategoryCreate, request: Request) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)

    _validate_category_name(body.name)
    require_len(body.description, MAX_DESCRIPTION, "The category description")

    with store.lock:
        if len(store.categories) >= MAX_CATEGORIES:
            raise bad_request(
                f"The container has reached the maximum of {MAX_CATEGORIES} tag categories."
            )

        name = _normalise_category_name(body.name, settings)
        if store.category_by_name(name) is not None:
            # Verbatim from the documented 400. Tenable.io does not upsert.
            raise bad_request(
                "A category with the name you specified already exists.", code="duplicate"
            )

        stamp = rfc3339_millis(store.now())
        category = {
            "uuid": store.next_uuid(),
            "created_at": stamp,
            "created_by": settings.user,
            "updated_at": stamp,
            "updated_by": settings.user,
            "name": name,
            "description": body.description or "",
            "reserved": False,
        }
        store.categories[category["uuid"]] = category
        return _category_payload(category)


@router.get("/tags/categories")
async def list_tag_categories(request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        categories = sorted(store.categories.values(), key=lambda c: c["name"])
        return {
            "categories": [_category_payload(c) for c in categories],
            "pagination": pagination(len(categories)),
        }


@router.get("/tags/categories/{uuid}")
async def get_tag_category(uuid: str, request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        return _category_payload(_find_category(store, uuid))


@router.put("/tags/categories/{uuid}")
async def update_tag_category(
    uuid: str, body: TagCategoryUpdate, request: Request
) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)

    _validate_category_name(body.name)
    require_len(body.description, MAX_DESCRIPTION, "The category description")

    with store.lock:
        category = _find_category(store, uuid)
        name = _normalise_category_name(body.name, settings)

        clash = store.category_by_name(name)
        if clash is not None and clash["uuid"] != uuid:
            raise bad_request(
                "A category with the name you specified already exists.", code="duplicate"
            )

        category["name"] = name
        category["description"] = resolve_on_omit(
            body,
            "description",
            category["description"],
            settings.quirks.on_omitted_description,
        )
        category["updated_at"] = rfc3339_millis(store.now())
        category["updated_by"] = settings.user
        return _category_payload(category)


@router.delete("/tags/categories/{uuid}")
async def delete_tag_category(uuid: str, request: Request) -> None:
    store = get_store(request)
    with store.lock:
        _find_category(store, uuid)
        # Deleting a category takes its values with it.
        for value_uuid in [
            v["uuid"] for v in store.tag_values.values() if v["category_uuid"] == uuid
        ]:
            del store.tag_values[value_uuid]
        del store.categories[uuid]


# -- value helpers ---------------------------------------------------------


def _value_payload(store: Store, tag: dict[str, Any]) -> dict[str, Any]:
    """Render a stored tag value the way the API reports it.

    ``category_name`` and ``category_description`` are resolved from the
    category on every read rather than copied at write time, so a category
    rename shows up in its values immediately -- as it does in production.
    """
    category = store.categories.get(tag["category_uuid"], {})
    payload: dict[str, Any] = {
        "uuid": tag["uuid"],
        "created_at": tag["created_at"],
        "created_by": tag["created_by"],
        "updated_at": tag["updated_at"],
        "updated_by": tag["updated_by"],
        "category_uuid": tag["category_uuid"],
        "value": tag["value"],
        "description": tag["description"],
        "type": tag["type"],
        "category_name": category.get("name", ""),
        "category_description": category.get("description", ""),
        "access_control": access_control(),
    }
    if tag.get("asset_filter"):
        # The response shape: a JSON-formatted *string*, not an object.
        payload["filters"] = {"asset": tag["asset_filter"]}
    return payload


def _resolve_category_for_value(
    store: Store, body: TagValueCreate, settings: Settings
) -> dict[str, Any]:
    """Find or create the category a new tag value belongs to.

    Unlike ``POST /tags/categories``, this path creates a category on demand
    when handed a ``category_name`` that does not exist yet. ``category_uuid``
    must already resolve.
    """
    has_uuid = bool(body.category_uuid)
    has_name = bool(body.category_name)

    if has_uuid and has_name:
        raise bad_request(
            "Specify either category_uuid or category_name, not both."
        )
    if not has_uuid and not has_name:
        raise bad_request("Either category_uuid or category_name is required.")

    if has_uuid:
        category = store.categories.get(body.category_uuid or "")
        if category is None:
            raise bad_request(
                f"The specified category {body.category_uuid} does not exist.",
                code="not_found",
            )
        return category

    name = _normalise_category_name(body.category_name or "", settings)
    _validate_category_name(name)
    existing = store.category_by_name(name)
    if existing is not None:
        return existing

    require_len(body.category_description, MAX_DESCRIPTION, "The category description")
    if len(store.categories) >= MAX_CATEGORIES:
        raise bad_request(
            f"The container has reached the maximum of {MAX_CATEGORIES} tag categories."
        )
    stamp = rfc3339_millis(store.now())
    category = {
        "uuid": store.next_uuid(),
        "created_at": stamp,
        "created_by": settings.user,
        "updated_at": stamp,
        "updated_by": settings.user,
        "name": name,
        "description": body.category_description or "",
        "reserved": False,
    }
    store.categories[category["uuid"]] = category
    return category


# -- values ----------------------------------------------------------------


@router.post("/tags/values")
async def create_tag_value(body: TagValueCreate, request: Request) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)

    if not body.value or not body.value.strip():
        raise bad_request("The tag value is required.")
    require_len(body.description, MAX_DESCRIPTION, "The tag description")

    with store.lock:
        category = _resolve_category_for_value(store, body, settings)

        if store.tag_value_by_value(category["uuid"], body.value) is not None:
            raise bad_request(
                "A tag with the category and value you specified already exists.",
                code="duplicate",
            )
        if len(store.values_in_category(category["uuid"])) >= MAX_VALUES_PER_CATEGORY:
            raise bad_request(
                f"The category has reached the maximum of "
                f"{MAX_VALUES_PER_CATEGORY} values."
            )

        asset_filter = ""
        if body.filters is not None:
            asset = body.filters.get("asset")
            if asset is None:
                raise bad_request("'filters' requires an 'asset' object.")
            asset_filter = encode_asset_filter(asset)

        stamp = rfc3339_millis(store.now())
        tag = {
            "uuid": store.next_uuid(),
            "created_at": stamp,
            "created_by": settings.user,
            "updated_at": stamp,
            "updated_by": settings.user,
            "category_uuid": category["uuid"],
            "value": body.value,
            "description": body.description or "",
            # The presence of filters is what makes a tag dynamic.
            "type": "dynamic" if asset_filter else "static",
            "asset_filter": asset_filter,
        }
        store.tag_values[tag["uuid"]] = tag
        return _value_payload(store, tag)


@router.get("/tags/values")
async def list_tag_values(request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        values = sorted(store.tag_values.values(), key=lambda v: v["value"])
        return {
            "values": [_value_payload(store, v) for v in values],
            "pagination": pagination(len(values), sort_by="value"),
        }


@router.get("/tags/values/{uuid}")
async def get_tag_value(uuid: str, request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        tag = store.tag_values.get(uuid)
        if tag is None:
            raise not_found(f"Tag value {uuid} was not found.")
        return _value_payload(store, tag)


@router.put("/tags/values/{uuid}")
async def update_tag_value(
    uuid: str, body: TagValueUpdate, request: Request
) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)
    require_len(body.description, MAX_DESCRIPTION, "The tag description")

    with store.lock:
        tag = store.tag_values.get(uuid)
        if tag is None:
            raise not_found(f"Tag value {uuid} was not found.")

        if was_provided(body, "value"):
            if not body.value or not body.value.strip():
                raise bad_request("The tag value cannot be empty.")
            clash = store.tag_value_by_value(tag["category_uuid"], body.value)
            if clash is not None and clash["uuid"] != uuid:
                raise bad_request(
                    "A tag with the category and value you specified already exists.",
                    code="duplicate",
                )
            tag["value"] = body.value

        tag["description"] = resolve_on_omit(
            body, "description", tag["description"], settings.quirks.on_omitted_description
        )

        if was_provided(body, "filters") and body.filters is not None:
            asset = body.filters.get("asset")
            if asset is None:
                raise bad_request("'filters' requires an 'asset' object.")
            tag["asset_filter"] = encode_asset_filter(asset)
        else:
            # Omitted filters: the docs do not say whether the rules survive.
            tag["asset_filter"] = resolve_on_omit(
                body, "filters", tag["asset_filter"], settings.quirks.on_omitted_filters
            )

        tag["type"] = "dynamic" if tag["asset_filter"] else "static"
        tag["updated_at"] = rfc3339_millis(store.now())
        tag["updated_by"] = settings.user
        return _value_payload(store, tag)


@router.delete("/tags/values/{uuid}")
async def delete_tag_value(uuid: str, request: Request) -> None:
    store = get_store(request)
    with store.lock:
        if uuid not in store.tag_values:
            raise not_found(f"Tag value {uuid} was not found.")
        del store.tag_values[uuid]


# -- asset filter catalogue ------------------------------------------------


@router.get("/tags/assets/filters")
async def list_asset_tag_filters() -> dict[str, Any]:
    return {"filters": ASSET_TAG_FILTERS}
