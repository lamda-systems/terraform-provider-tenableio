"""Runtime configuration for the mock.

Everything here is driven by environment variables so the same image can be
pointed at different behaviours from a compose file or a CI job without a
rebuild. All quirks default to off: the out-of-the-box server is the strict,
literal-echo one.
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field
from enum import Enum


class OnOmit(str, Enum):
    """What an update does when a nullable key is absent from the request body.

    Tenable.io documents ``description`` and ``filters`` as optional on ``PUT``
    but never says whether leaving one out preserves or clears the stored
    value. Both readings are defensible -- ``PUT`` implies whole-object
    replacement, while the API's tolerance of partial bodies implies a merge --
    so the mock refuses to guess and makes it selectable.

    This is not a hypothetical. A provider that declares ``description`` with a
    ``""`` default but serialises it with ``omitempty`` never puts the key on
    the wire when a user clears it. Under :attr:`PRESERVES` the server then
    echoes the stale text and Terraform aborts the apply with "Provider produced
    inconsistent result after apply". Under :attr:`CLEARS` the same provider
    looks fine. A correct provider passes both ways, which is exactly why both
    are offered.
    """

    CLEARS = "clears"
    PRESERVES = "preserves"


class ScanUpdateEcho(str, Enum):
    """What ``PUT /scans/{id}`` answers with.

    Tenable.io documents the response as the scan object *bare* -- not wrapped
    in ``"scan"`` the way ``POST /scans`` wraps it -- but live tenants have been
    observed answering with no body at all, and the docs also say the 200 has no
    content in places. The two readings matter because the update echo is the
    *only* response that reports a scan's description: ``GET /scans/{id}`` never
    does. A provider that needs the echo to verify what it wrote must therefore
    also cope with not getting one.
    """

    EMPTY = "empty"
    OBJECT = "object"


@dataclass(frozen=True)
class Quirks:
    """Behaviours that are undocumented, or documented inconsistently.

    Each one is off by default. Turn them on to prove the provider tolerates
    them rather than to make the mock more convenient.
    """

    on_omitted_description: OnOmit = OnOmit.CLEARS

    #: What ``PUT /tags/values`` does when the body carries no ``filters``.
    #:
    #: Unresolved in the docs, and it is the reason the provider forces a
    #: replacement rather than an update when filters are removed from a
    #: dynamic tag. Under :attr:`OnOmit.CLEARS` the tag reverts to static;
    #: under :attr:`OnOmit.PRESERVES` it stays dynamic with its old rules.
    on_omitted_filters: OnOmit = OnOmit.CLEARS

    #: Fold tag category names to lower case on write and echo the folded form.
    #:
    #: The documented example for ``POST /tags/values`` sends
    #: ``"category_name": "Location"`` and gets back ``"category_name":
    #: "location"``, which suggests live Tenable.io normalises. Nothing states
    #: it in prose, so it is a quirk rather than the default. Any provider
    #: attribute that writes an echoed category name straight into Terraform
    #: state fails its apply when this is on.
    lowercase_category_names: bool = False

    #: Report ``description`` in the ``info`` object of ``GET /scans/{id}``.
    #:
    #: Live Tenable.io does not: the scan details endpoint describes a scan
    #: *result*, and its documented ``info`` schema has no ``description`` at
    #: all. The mock used to echo one, which is why a provider that read its
    #: state back from that endpoint passed here and then reported every
    #: described scan as having had its description wiped in production. Off is
    #: the faithful setting; turn it on only to prove the provider also accepts
    #: a tenant that does report it.
    scan_details_description: bool = False

    #: What ``PUT /scans/{id}`` returns. See :class:`ScanUpdateEcho`.
    scan_update_echo: ScanUpdateEcho = ScanUpdateEcho.EMPTY

    #: Reject request bodies carrying fields the endpoint does not define.
    #:
    #: Real Tenable.io ignores unknown fields. This is a lint mode for catching
    #: provider typos, not a fidelity setting -- leave it off when the goal is
    #: to emulate production.
    reject_unknown_fields: bool = False


@dataclass(frozen=True)
class Settings:
    #: When set, the ``X-ApiKeys`` header must carry exactly these. Left empty,
    #: any syntactically valid header is accepted, which is what most
    #: acceptance tests want.
    access_key: str = ""
    secret_key: str = ""

    #: Principal recorded as ``created_by`` / ``updated_by`` / ``owner``.
    user: str = "terraform@example.com"

    #: Populate the read-only endpoints (scanners, workbench assets) and the
    #: default network and folders that a real container always exposes.
    seed: bool = True

    #: Freeze the clock so responses are byte-reproducible. Set
    #: ``MOCK_FROZEN_CLOCK=0`` for a server whose timestamps advance.
    frozen_clock: bool = True

    quirks: Quirks = field(default_factory=Quirks)


def _env_bool(name: str, default: bool) -> bool:
    raw = os.environ.get(name)
    if raw is None or raw == "":
        return default
    return raw.strip().lower() in {"1", "true", "yes", "on"}


def settings_from_env(env: dict[str, str] | None = None) -> Settings:
    """Build :class:`Settings` from the environment.

    Recognised variables, all optional:

    ``MOCK_ACCESS_KEY`` / ``MOCK_SECRET_KEY``
        Require these exact credentials.
    ``MOCK_USER``
        Principal recorded on created/updated objects.
    ``MOCK_SEED``
        Seed the read-only endpoints. Default on.
    ``MOCK_FROZEN_CLOCK``
        Freeze timestamps for reproducibility. Default on.
    ``MOCK_OMITTED_DESCRIPTION``
        ``clears`` (default) or ``preserves``.
    ``MOCK_OMITTED_FILTERS``
        ``clears`` (default) or ``preserves``.
    ``MOCK_SCAN_DETAILS_DESCRIPTION``
        Report ``description`` on ``GET /scans/{id}``. Default off, which is
        what live Tenable.io does.
    ``MOCK_SCAN_UPDATE_ECHO``
        ``empty`` (default) or ``object``.
    ``MOCK_LOWERCASE_CATEGORY_NAMES``
        Default off.
    ``MOCK_REJECT_UNKNOWN_FIELDS``
        Default off.
    """
    if env is not None:
        prior = dict(os.environ)
        os.environ.update(env)
        try:
            return settings_from_env()
        finally:
            os.environ.clear()
            os.environ.update(prior)

    omitted = _env_on_omit("MOCK_OMITTED_DESCRIPTION")
    omitted_filters = _env_on_omit("MOCK_OMITTED_FILTERS")

    return Settings(
        access_key=os.environ.get("MOCK_ACCESS_KEY", ""),
        secret_key=os.environ.get("MOCK_SECRET_KEY", ""),
        user=os.environ.get("MOCK_USER", "") or "terraform@example.com",
        seed=_env_bool("MOCK_SEED", True),
        frozen_clock=_env_bool("MOCK_FROZEN_CLOCK", True),
        quirks=Quirks(
            on_omitted_description=omitted,
            on_omitted_filters=omitted_filters,
            scan_details_description=_env_bool("MOCK_SCAN_DETAILS_DESCRIPTION", False),
            scan_update_echo=_env_scan_update_echo(),
            lowercase_category_names=_env_bool("MOCK_LOWERCASE_CATEGORY_NAMES", False),
            reject_unknown_fields=_env_bool("MOCK_REJECT_UNKNOWN_FIELDS", False),
        ),
    )


def _env_scan_update_echo() -> ScanUpdateEcho:
    name = "MOCK_SCAN_UPDATE_ECHO"
    raw = os.environ.get(name, "").strip().lower()
    if not raw:
        return ScanUpdateEcho.EMPTY
    try:
        return ScanUpdateEcho(raw)
    except ValueError as exc:
        valid = ", ".join(sorted(e.value for e in ScanUpdateEcho))
        raise ValueError(f"{name} must be one of: {valid} (got {raw!r})") from exc


def _env_on_omit(name: str) -> OnOmit:
    raw = os.environ.get(name, "").strip().lower()
    if not raw:
        return OnOmit.CLEARS
    try:
        return OnOmit(raw)
    except ValueError as exc:
        valid = ", ".join(sorted(o.value for o in OnOmit))
        raise ValueError(f"{name} must be one of: {valid} (got {raw!r})") from exc
