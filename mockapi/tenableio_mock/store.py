"""In-memory state for the mock.

Identifiers are deterministic on purpose: a fixed sequence of requests always
produces the same UUIDs and integer IDs, so acceptance tests can assert on
concrete values and failures are reproducible. Nothing here is persisted --
restarting the process, or calling ``POST /__mock/reset``, gives a clean
container.
"""

from __future__ import annotations

import threading
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any

#: The instant a frozen clock reports. Arbitrary but fixed.
FROZEN_NOW = datetime(2026, 1, 1, tzinfo=timezone.utc)


def rfc3339_millis(moment: datetime) -> str:
    """Format as the tags endpoints do: RFC 3339 with milliseconds."""
    return moment.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.") + (
        f"{moment.microsecond // 1000:03d}Z"
    )


def unix_seconds(moment: datetime) -> int:
    """Format as the scan, network, exclusion and agent group endpoints do."""
    return int(moment.astimezone(timezone.utc).timestamp())


@dataclass
class Record:
    """One request the server handled."""

    method: str
    path: str
    query: str
    status: int
    at: str
    body: Any = None
    #: Keys physically present in the request body. Distinct from which keys
    #: hold a truthy value -- that difference is how an ``omitempty`` that
    #: silently drops a meaningful empty string gets caught.
    body_keys: list[str] = field(default_factory=list)

    def as_dict(self) -> dict[str, Any]:
        return {
            "method": self.method,
            "path": self.path,
            "query": self.query,
            "status": self.status,
            "at": self.at,
            "body": self.body,
            "body_keys": self.body_keys,
        }


class Store:
    """Every object the mock knows about, behind one lock."""

    def __init__(self, frozen_clock: bool = True) -> None:
        self._frozen = frozen_clock
        self.lock = threading.RLock()
        self.reset()

    # -- lifecycle ---------------------------------------------------------

    def reset(self) -> None:
        with self.lock:
            self.categories: dict[str, dict[str, Any]] = {}
            self.tag_values: dict[str, dict[str, Any]] = {}
            self.folders: dict[int, dict[str, Any]] = {}
            self.networks: dict[str, dict[str, Any]] = {}
            self.exclusions: dict[int, dict[str, Any]] = {}
            self.agent_groups: dict[int, dict[str, Any]] = {}
            self.policies: dict[int, dict[str, Any]] = {}
            self.scans: dict[int, dict[str, Any]] = {}
            self.scanners: dict[int, dict[str, Any]] = {}
            self.assets: dict[str, dict[str, Any]] = {}
            self.requests: list[Record] = []
            self._uuid_seq = 0
            self._id_seq = 0

    def now(self) -> datetime:
        return FROZEN_NOW if self._frozen else datetime.now(timezone.utc)

    # -- identifiers -------------------------------------------------------

    def next_uuid(self) -> str:
        """A deterministic, syntactically valid UUID.

        The sequence number lands in the final group so identifiers stay
        readable in logs and stable across runs.
        """
        self._uuid_seq += 1
        return f"00000000-0000-4000-8000-{self._uuid_seq:012d}"

    def next_id(self) -> int:
        """A deterministic integer ID.

        One counter is shared across folders, scans, policies, exclusions and
        agent groups so an ID is never ambiguous about which object it names,
        which makes a failing assertion easier to read.
        """
        self._id_seq += 1
        return self._id_seq

    # -- lookups -----------------------------------------------------------

    def category_by_name(self, name: str) -> dict[str, Any] | None:
        """Resolve a category by exact name; names are unique per container."""
        for category in self.categories.values():
            if category["name"] == name:
                return category
        return None

    def tag_value_by_value(self, category_uuid: str, value: str) -> dict[str, Any] | None:
        """Resolve a value within a category.

        The uniqueness constraint Tenable.io enforces is on the (category,
        value) pair, not on the value alone.
        """
        for tag in self.tag_values.values():
            if tag["category_uuid"] == category_uuid and tag["value"] == value:
                return tag
        return None

    def values_in_category(self, category_uuid: str) -> list[dict[str, Any]]:
        return [t for t in self.tag_values.values() if t["category_uuid"] == category_uuid]

    # -- request log -------------------------------------------------------

    def add_record(self, record: Record) -> None:
        with self.lock:
            self.requests.append(record)

    def snapshot_requests(self) -> list[Record]:
        with self.lock:
            return list(self.requests)

    def clear_requests(self) -> None:
        with self.lock:
            self.requests = []
