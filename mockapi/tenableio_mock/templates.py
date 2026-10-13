"""The Tenable-provided scan templates.

Read-only and identical in every container, so they are a fixed catalogue
rather than store state: ``/__mock/reset`` and ``MOCK_SEED`` do not touch them.

Two deliberate choices, both made because they are stricter than the docs
require rather than because production is known to behave this way:

* **Each template list hands out its own uuids.** The same template appears in
  the ``scan`` and ``policy`` lists, and the docs never say its uuid is the same
  in both. Giving each list distinct uuids is the reading under which a
  provider that takes a policy uuid from the scan list fails, which is the
  mistake worth catching.
* **Titles are not unique.** A network template and its agent counterpart share
  a title ("Malware Scan", "Policy Compliance Auditing"), as in the documented
  example. A lookup by title has to cope with that.

The uuids are synthetic but keep Tenable's shape: a 36-character uuid followed
by a 16-character hex suffix.

The details returned for each template are a placeholder skeleton -- enough for
the response *shape* to be right. They are not a faithful copy of any
template's settings.
"""

from __future__ import annotations

import uuid
from typing import Any

#: The template lists ``/editor/{type}/templates`` serves.
TEMPLATE_TYPES = ("scan", "policy", "remediation")

#: Every list entry carries an ``icon``: a base64-encoded image for the UI. It is
#: undocumented, but a live tenant returns it, so the mock does too. A tiny
#: placeholder keeps responses small; the provider ignores the field.
_ICON = "PD94bWwgdmVyc2lvbj0iMS4wIj8+"

_NAMESPACE = uuid.UUID("5e1f3c0a-7d2b-4c8e-9a61-2f4b8d0c9e17")
_SUFFIX = "d24bd260ef5f9e66"

# name, title, desc, order, is_agent, flags, lists it appears in
_CATALOGUE: list[tuple[str, str, str, int | None, bool | None, dict[str, bool], tuple[str, ...]]] = [
    ("advanced", "Advanced Network Scan", "Configure a scan without using any recommendations.",
     1, None, {}, ("scan", "policy", "remediation")),
    ("basic", "Basic Network Scan", "A full system scan suitable for any host.",
     2, None, {}, ("scan", "policy", "remediation")),
    ("discovery", "Host Discovery", "A simple scan to discover live hosts and open ports.",
     3, None, {}, ("scan", "policy")),
    ("patch_audit", "Credentialed Patch Audit", "Authenticate to hosts and enumerate missing updates.",
     4, None, {}, ("scan", "policy")),
    ("compliance", "Policy Compliance Auditing", "Audit system configurations against a known baseline.",
     5, None, {"subscription_only": True}, ("scan", "policy")),
    ("scap", "SCAP and OVAL Auditing", "Audit systems using SCAP and OVAL definitions.",
     6, None, {"subscription_only": True}, ("scan", "policy")),
    ("malware", "Malware Scan", "Scan for malware on Windows and Unix systems.",
     7, None, {}, ("scan", "policy")),
    ("webapp", "Web Application Tests", "Scan for published and unknown web vulnerabilities.",
     8, None, {}, ("scan", "policy")),
    ("pci", "Internal PCI Network Scan", "Perform an internal PCI DSS vulnerability scan.",
     9, None, {}, ("scan", "policy")),
    ("asv", "PCI Quarterly External Scan", "Approved for quarterly external scanning as required by PCI.",
     10, None, {"cloud_only": True, "subscription_only": True}, ("scan",)),
    ("mdm", "MDM Config Audit", "Audit the configuration of mobile device managers.",
     11, None, {"unsupported": True}, ("scan", "policy")),
    ("agent_advanced", "Advanced Agent Scan", "Configure an agent scan without using any recommendations.",
     None, True, {}, ("scan", "policy")),
    ("agent_basic", "Basic Agent Scan", "Scan systems connected via Nessus Agents.",
     None, True, {}, ("scan", "policy")),
    ("agent_compliance", "Policy Compliance Auditing", "Audit systems connected via Nessus Agents.",
     None, True, {"subscription_only": True}, ("scan", "policy")),
    ("agent_malware", "Malware Scan", "Scan for malware on systems connected via Nessus Agents.",
     None, True, {}, ("scan", "policy")),
    ("agent_scap", "SCAP and OVAL Auditing", "Audit systems using SCAP and OVAL definitions via agents.",
     None, True, {"subscription_only": True}, ("scan", "policy")),
]


def template_uuid(template_type: str, name: str) -> str:
    """The uuid ``name`` has in the ``template_type`` list. Stable across runs."""
    return f"{uuid.uuid5(_NAMESPACE, f'{template_type}/{name}')}{_SUFFIX}"


def _entry(template_type: str, row: tuple) -> dict[str, Any]:
    name, title, desc, order, is_agent, flags, _ = row
    return {
        "uuid": template_uuid(template_type, name),
        "name": name,
        "title": title,
        "desc": desc,
        "order": order,
        "unsupported": flags.get("unsupported", False),
        "cloud_only": flags.get("cloud_only", False),
        "subscription_only": flags.get("subscription_only", False),
        "manager_only": flags.get("manager_only", False),
        # Documented as "boolean or null"; null for anything that is not an
        # agent template, as in the documented example.
        "is_agent": is_agent,
        "is_was": None,
        "icon": _ICON,
    }


def list_templates(template_type: str) -> list[dict[str, Any]]:
    return [_entry(template_type, row) for row in _CATALOGUE if template_type in row[6]]


def find_template(template_type: str, template_uuid_: str) -> dict[str, Any] | None:
    for entry in list_templates(template_type):
        if entry["uuid"] == template_uuid_:
            return entry
    return None


def template_details(entry: dict[str, Any]) -> dict[str, Any]:
    """The ``GET /editor/{type}/templates/{uuid}`` body for one template.

    Shaped like the documented response; the settings are a placeholder.
    """
    settings: dict[str, Any] = {
        "basic": {
            "title": "Basic",
            "inputs": [
                {"type": "entry", "name": "Name", "id": "name", "required": True},
                {"type": "textarea", "name": "Description", "id": "description"},
            ],
            "groups": [{"title": "Permissions", "name": "permissions"}],
        },
        "report": {
            "title": "Report",
            "sections": [
                {
                    "title": "Output",
                    "name": "report_output",
                    "inputs": [
                        {
                            "type": "checkbox",
                            "id": "reverse_lookup",
                            "label": "Designate hosts by their DNS name",
                            "default": "no",
                        }
                    ],
                }
            ],
        },
    }
    details: dict[str, Any] = {
        "is_was": entry["is_was"],
        "user_permissions": None,
        "owner": None,
        "title": entry["title"],
        "is_agent": entry["is_agent"],
        "uuid": entry["uuid"],
        "name": entry["name"],
        "settings": settings,
        "filter_attributes": [],
        "credentials": {},
        "compliance": {},
    }
    if entry["name"] in ("advanced", "agent_advanced"):
        # Only templates that expose plugin selection carry plugin families.
        details["plugins"] = {
            "families": {
                "Backdoors": {"id": 1, "count": 100, "status": "enabled"},
                "Web Servers": {"id": 4, "count": 1000, "status": "enabled"},
            }
        }
    return details
