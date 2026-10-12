"""Scan policies -- what the Tenable.io UI calls user-defined scan templates.

Three shapes are in play and they do not match each other, which is faithful
to the documentation:

* ``POST /policies`` takes ``{uuid, settings, plugins}``, where ``settings`` is
  one flat object holding the name, description and template settings side by
  side, and returns only ``{policy_id, policy_name}``.
* ``GET /policies/{id}`` returns ``{uuid, settings, plugins}`` with every
  setting a string -- except ``max_compliance_check_timeout`` and
  ``http_reauth_delay``, which come back as integers -- and no metadata at all.
* ``GET /policies`` returns flat summaries: the metadata, none of the settings.
* ``PUT /policies/{id}`` returns an empty body.

The ``uuid`` in a request and in the details is the *template* the policy is
based on, taken from the ``policy`` template list.

What the docs settle, the mock does: an update preserves every setting it
leaves out, a setting the template does not define is discarded without an
error, and only templates that offer plugin selection accept ``plugins``.
"""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Request, Response

from ..config import PolicyDetailsSettings, Settings
from ..errors import bad_request, not_found
from ..models import PolicyCreate, PolicyUpdate
from ..store import Store, unix_seconds
from ..template_settings import AGENT_SETTINGS, DEPENDENT_SETTINGS, NETWORK_SETTINGS
from ..templates import find_template
from ._common import get_settings, get_store, reject_unknown

router = APIRouter(tags=["policies"])

MAX_DESCRIPTION = 3000
VISIBILITIES = {"private", "shared"}

#: Settings the documented details response returns as integers.
INTEGER_SETTINGS = {"max_compliance_check_timeout", "http_reauth_delay"}

#: Templates that offer plugin selection, and the families they list. The
#: provider does not validate family names, so the mock does, as the stricter
#: reading.
PLUGIN_TEMPLATES = {"advanced", "agent_advanced"}
PLUGIN_FAMILIES = (
    "Backdoors", "CGI abuses", "Denial of Service", "General", "Misc.", "SCADA",
    "SMTP problems", "Ubuntu Local Security Checks", "Web Servers", "Windows",
)
PLUGIN_STATUSES = {"enabled", "disabled"}


def _find(store: Store, policy_id: int) -> dict[str, Any]:
    policy = store.policies.get(policy_id)
    if policy is None:
        raise not_found(f"Policy {policy_id} was not found.")
    return policy


def _defined_settings(template: dict[str, Any]) -> dict[str, Any]:
    return AGENT_SETTINGS if template["is_agent"] else NETWORK_SETTINGS


def _check_name(name: Any) -> None:
    if not isinstance(name, str) or not name.strip():
        raise bad_request("The policy name is required.")


def _apply_settings(policy: dict[str, Any], template: dict[str, Any], settings: dict[str, Any]) -> None:
    """Merge request settings into a policy. Keys not sent are left alone."""
    defined = _defined_settings(template)
    for key, value in settings.items():
        if key == "name":
            _check_name(value)
            policy["name"] = value
        elif key == "description":
            if not isinstance(value, str):
                raise bad_request("description must be a string.")
            if len(value) > MAX_DESCRIPTION:
                raise bad_request(
                    f"The policy description exceeds the maximum length of "
                    f"{MAX_DESCRIPTION} characters."
                )
            policy["description"] = value
        elif key == "visibility":
            # Not a documented policy setting; accepted because the deprecated
            # tenableio_policy has always sent it, and reported by the list.
            if value not in VISIBILITIES:
                raise bad_request(f"visibility must be one of: {', '.join(sorted(VISIBILITIES))}.")
            policy["visibility"] = value
        elif key in defined:
            if isinstance(value, bool) or not isinstance(value, (str, int)):
                raise bad_request(f"Setting {key!r} must be a string or an integer.")
            policy["settings"][key] = str(value)
        # Anything else is a setting the template does not define, which the
        # docs say is discarded -- silently, not with an error.


def _apply_plugins(policy: dict[str, Any], template: dict[str, Any], plugins: dict[str, Any]) -> None:
    if template["name"] not in PLUGIN_TEMPLATES:
        raise bad_request(f"The {template['title']} template does not support plugin selection.")
    for family, entry in plugins.items():
        if family not in PLUGIN_FAMILIES:
            raise bad_request(f"Unknown plugin family {family!r}.")
        status = entry.get("status") if isinstance(entry, dict) else None
        if status not in PLUGIN_STATUSES:
            raise bad_request(f"Plugin family {family!r} needs a status of enabled or disabled.")
        policy["plugins"][family] = status


def _reported_settings(policy: dict[str, Any], settings: Settings) -> dict[str, Any]:
    reported: dict[str, Any] = {"name": policy["name"], "description": policy["description"]}
    stored = policy["settings"]
    omit_inactive = settings.quirks.policy_details_settings is PolicyDetailsSettings.OMIT_INACTIVE
    for key, value in stored.items():
        toggle = DEPENDENT_SETTINGS.get(key)
        if omit_inactive and toggle is not None and stored.get(toggle) != "yes":
            continue
        if key in INTEGER_SETTINGS and value.isdigit():
            reported[key] = int(value)
        else:
            reported[key] = value
    return reported


def _summary(policy: dict[str, Any]) -> dict[str, Any]:
    """A policy as ``GET /policies`` documents it."""
    return {
        "id": policy["id"],
        "template_uuid": policy["template_uuid"],
        "name": policy["name"],
        "description": policy["description"],
        "owner": policy["owner"],
        "owner_id": policy["owner_id"],
        "visibility": policy["visibility"],
        "shared": 1 if policy["visibility"] == "shared" else 0,
        "user_permissions": 128,
        "creation_date": policy["creation_date"],
        "last_modification_date": policy["last_modification_date"],
        "no_target": "false",
    }


@router.post("/policies")
async def create_policy(body: PolicyCreate, request: Request) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)
    template = find_template("policy", body.uuid)
    if template is None:
        raise bad_request(f"Unknown template {body.uuid!r}: use a uuid from the policy template list.")
    _check_name(body.settings.get("name"))

    with store.lock:
        stamp = unix_seconds(store.now())
        policy: dict[str, Any] = {
            "id": store.next_id(),
            "template_uuid": body.uuid,
            "name": "",
            "description": "",
            "settings": {k: str(v) for k, v in _defined_settings(template).items()},
            "plugins": {f: "enabled" for f in PLUGIN_FAMILIES} if template["name"] in PLUGIN_TEMPLATES else None,
            "owner": settings.user,
            "owner_id": 1,
            "visibility": "private",
            "creation_date": stamp,
            "last_modification_date": stamp,
        }
        _apply_settings(policy, template, body.settings)
        if body.plugins:
            _apply_plugins(policy, template, body.plugins)
        store.policies[policy["id"]] = policy
        return {"policy_id": policy["id"], "policy_name": policy["name"]}


@router.get("/policies")
async def list_policies(request: Request) -> dict[str, Any]:
    store = get_store(request)
    with store.lock:
        return {"policies": [_summary(p) for p in store.policies.values()]}


@router.get("/policies/{policy_id}")
async def get_policy(policy_id: int, request: Request) -> dict[str, Any]:
    store, settings = get_store(request), get_settings(request)
    with store.lock:
        policy = _find(store, policy_id)
        details: dict[str, Any] = {
            "uuid": policy["template_uuid"],
            "settings": _reported_settings(policy, settings),
        }
        if policy["plugins"] is not None:
            details["plugins"] = {f: {"status": s} for f, s in policy["plugins"].items()}
        return details


@router.put("/policies/{policy_id}")
async def update_policy(policy_id: int, body: PolicyUpdate, request: Request) -> Response:
    store, settings = get_store(request), get_settings(request)
    reject_unknown(body, settings)

    with store.lock:
        policy = _find(store, policy_id)
        # uuid is ignored: a policy cannot move to another template.
        template = find_template("policy", policy["template_uuid"])
        assert template is not None
        if body.settings:
            _apply_settings(policy, template, body.settings)
        if body.plugins:
            _apply_plugins(policy, template, body.plugins)
        policy["last_modification_date"] = unix_seconds(store.now())
    return Response(status_code=200)


@router.delete("/policies/{policy_id}")
async def delete_policy(policy_id: int, request: Request) -> Response:
    store = get_store(request)
    with store.lock:
        _find(store, policy_id)
        del store.policies[policy_id]
    return Response(status_code=200)
