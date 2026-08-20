"""Baseline objects a real Tenable.io container always has.

Seeding covers only what cannot be created through the API -- scanners and
workbench assets -- plus the system folders and default network that exist
before any Terraform runs. Tag categories and tag values are deliberately *not*
seeded: a test that wants one should create it, so that the request which
created it is visible in the recording.
"""

from __future__ import annotations

from .config import Settings
from .store import Store, rfc3339_millis, unix_seconds


def seed_store(store: Store, settings: Settings) -> None:
    with store.lock:
        _seed_folders(store)
        _seed_network(store, settings)
        _seed_scanners(store)
        _seed_assets(store)


def _seed_folders(store: Store) -> None:
    """The two folders Tenable.io creates per container.

    ``custom: 0`` marks them as system-owned, which is what makes them
    un-renamable and un-deletable.
    """
    for name, kind in (("My Scans", "main"), ("Trash", "trash")):
        folder_id = store.next_id()
        store.folders[folder_id] = {
            "id": folder_id,
            "name": name,
            "type": kind,
            "custom": 0,
            "unread_count": 0,
            "default_tag": 1 if kind == "main" else 0,
        }


def _seed_network(store: Store, settings: Settings) -> None:
    stamp = unix_seconds(store.now())
    uuid = store.next_uuid()
    store.networks[uuid] = {
        "uuid": uuid,
        "name": "Default",
        "description": "The default network object.",
        "is_default": True,
        "created_by": settings.user,
        "created_in_seconds": stamp,
        "modified_in_seconds": stamp,
        "scanner_count": 1,
        "assets_ttl_days": 180,
    }


def _seed_scanners(store: Store) -> None:
    stamp = unix_seconds(store.now())
    scanners = [
        {
            "id": 100001,
            "name": "US Cloud Scanner",
            "type": "managed",
            "pool": True,
            "network_name": "Default",
        },
        {
            "id": 100002,
            "name": "EU Cloud Scanner",
            "type": "managed",
            "pool": True,
            "network_name": "Default",
        },
    ]
    for entry in scanners:
        store.scanners[entry["id"]] = {
            "id": entry["id"],
            "uuid": store.next_uuid(),
            "name": entry["name"],
            "type": entry["type"],
            "status": "on",
            "scan_count": 0,
            "engine_version": "10.8.3",
            "platform": "LINUX",
            "loaded_plugin_set": "202601010000",
            "owner": "system",
            "owner_id": 0,
            "pool": entry["pool"],
            "shared": 1,
            "user_permissions": 64,
            "creation_date": stamp,
            "last_modification_date": stamp,
            "network_name": entry["network_name"],
        }


def _seed_assets(store: Store) -> None:
    stamp = rfc3339_millis(store.now())
    assets = [
        {
            "id": "a1b2c3d4-0000-4000-8000-000000000001",
            "fqdn": ["web01.example.com"],
            "ipv4": ["10.0.1.10"],
            "operating_system": ["Ubuntu 24.04"],
            "netbios_name": ["WEB01"],
            "hostname": ["web01"],
            "has_agent": True,
        },
        {
            "id": "a1b2c3d4-0000-4000-8000-000000000002",
            "fqdn": ["db01.example.com"],
            "ipv4": ["10.0.2.20"],
            "operating_system": ["Microsoft Windows Server 2022"],
            "netbios_name": ["DB01"],
            "hostname": ["db01"],
            "has_agent": False,
        },
    ]
    for entry in assets:
        store.assets[entry["id"]] = {
            "id": entry["id"],
            "has_agent": entry["has_agent"],
            "has_plugin_results": True,
            "fqdn": entry["fqdn"],
            "ipv4": entry["ipv4"],
            "ipv6": [],
            "mac_address": [],
            "netbios_name": entry["netbios_name"],
            "operating_system": entry["operating_system"],
            "agent_name": [],
            "last_seen": stamp,
            "first_seen": stamp,
            "created_at": stamp,
            "updated_at": stamp,
            "system_type": "general-purpose",
            "hostname": entry["hostname"],
            "aws_ec2_instance_id": [],
            "aws_vpc_id": [],
            "azure_resource_id": [],
            "azure_vm_id": [],
            "gcp_project_id": [],
            "gcp_instance_id": [],
            "counts": {"critical": 0, "high": 1, "medium": 3, "low": 5, "info": 12},
        }
