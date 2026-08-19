"""FastAPI application factory."""

from __future__ import annotations

import json
from typing import Any, Awaitable, Callable

from fastapi import FastAPI, Request, Response, status
from fastapi.responses import JSONResponse

from .config import Settings, settings_from_env
from .errors import install_handlers, json_error
from .routers import (
    agent_groups,
    exclusions,
    folders,
    networks,
    policies,
    scans,
    scanners,
    tags,
    workbenches,
)
from .seed import seed_store
from .store import Record, Store, rfc3339_millis

#: Tenable.io caps tag requests at 1 MB. The mock applies the cap to every
#: endpoint, which is stricter than production but only in the direction that
#: surfaces provider bugs.
MAX_BODY_BYTES = 1 << 20

#: Administrative endpoints live under this prefix. They are not part of the
#: Tenable.io API and are exempt from authentication so a pipeline can always
#: reach them.
ADMIN_PREFIX = "/__mock/"

#: FastAPI's own documentation endpoints. Also not Tenable.io paths, so they
#: are exempt from the X-ApiKeys check and are never recorded -- a provider
#: will never call them, and requiring credentials would only make the
#: interactive schema browser useless.
META_PATHS = frozenset({"/openapi.json", "/docs", "/docs/oauth2-redirect", "/redoc"})


def _is_meta(path: str) -> bool:
    return path.startswith(ADMIN_PREFIX) or path in META_PATHS


def create_app(settings: Settings | None = None) -> FastAPI:
    settings = settings or settings_from_env()
    store = Store(frozen_clock=settings.frozen_clock)
    if settings.seed:
        seed_store(store, settings)

    app = FastAPI(
        title="Tenable.io mock API",
        version="1.0.0",
        description=(
            "In-memory fake of the Tenable.io endpoints the Terraform provider "
            "calls. Strict by default; see the Quirks settings for the "
            "behaviours that production leaves ambiguous."
        ),
    )
    app.state.settings = settings
    app.state.store = store

    install_handlers(app)
    _install_middleware(app, settings, store)

    for router in (
        tags.router,
        folders.router,
        networks.router,
        exclusions.router,
        agent_groups.router,
        scanners.router,
        policies.router,
        scans.router,
        workbenches.router,
    ):
        app.include_router(router)

    _install_admin(app, settings, store)
    return app


def _install_middleware(app: FastAPI, settings: Settings, store: Store) -> None:
    @app.middleware("http")
    async def capture_and_authenticate(
        request: Request, call_next: Callable[[Request], Awaitable[Response]]
    ) -> Response:
        is_meta = _is_meta(request.url.path)

        body = await request.body()
        if len(body) > MAX_BODY_BYTES:
            # 413; the Starlette constant for it was renamed, so use the literal.
            return json_error(413, "Request body exceeds the maximum size of 1 MB.")

        # Reading the body above exhausts the ASGI receive channel. Replay it
        # so the route handler downstream can still parse the request.
        async def replay() -> dict[str, Any]:
            return {"type": "http.request", "body": body, "more_body": False}

        request._receive = replay  # noqa: SLF001 - the documented Starlette workaround

        if not is_meta:
            failure = _authenticate(request, settings)
            if failure is not None:
                _record(store, request, body, failure.status_code)
                return failure

        response = await call_next(request)

        if not is_meta:
            _record(store, request, body, response.status_code)
        return response


def _authenticate(request: Request, settings: Settings) -> JSONResponse | None:
    """Validate the ``X-ApiKeys`` header.

    Tenable.io answers 401 for a missing or malformed header on every endpoint,
    so the check is applied uniformly rather than per route.
    """
    raw = request.headers.get("X-ApiKeys", "")
    if not raw:
        return json_error(status.HTTP_401_UNAUTHORIZED, "Missing X-ApiKeys header.")

    access, secret = _parse_api_keys(raw)
    if not access or not secret:
        return json_error(
            status.HTTP_401_UNAUTHORIZED,
            "Malformed X-ApiKeys header; expected accessKey=...;secretKey=...;",
        )
    if settings.access_key and access != settings.access_key:
        return json_error(status.HTTP_401_UNAUTHORIZED, "Invalid access key.")
    if settings.secret_key and secret != settings.secret_key:
        return json_error(status.HTTP_401_UNAUTHORIZED, "Invalid secret key.")
    return None


def _parse_api_keys(raw: str) -> tuple[str, str]:
    access = secret = ""
    for part in raw.split(";"):
        key, _, value = part.strip().partition("=")
        if key.strip() == "accessKey":
            access = value.strip()
        elif key.strip() == "secretKey":
            secret = value.strip()
    return access, secret


def _record(store: Store, request: Request, body: bytes, status_code: int) -> None:
    parsed: Any = None
    keys: list[str] = []
    if body:
        try:
            parsed = json.loads(body)
        except ValueError:
            parsed = None
        if isinstance(parsed, dict):
            keys = sorted(parsed)

    store.add_record(
        Record(
            method=request.method,
            path=request.url.path,
            query=request.url.query,
            status=status_code,
            at=rfc3339_millis(store.now()),
            body=parsed,
            body_keys=keys,
        )
    )


def _install_admin(app: FastAPI, settings: Settings, store: Store) -> None:
    """Endpoints for driving the mock from a test or a pipeline.

    Namespaced under ``/__mock/`` so they can never collide with a real
    Tenable.io path, and exempt from authentication.
    """

    @app.get(f"{ADMIN_PREFIX}health", tags=["mock"])
    async def health() -> dict[str, str]:
        return {"status": "ok"}

    @app.get(f"{ADMIN_PREFIX}requests", tags=["mock"])
    async def requests(method: str | None = None, path: str | None = None) -> dict[str, Any]:
        """Every request handled so far, oldest first.

        ``body_keys`` on each record lists the keys physically present in the
        request body. Asserting that an expected key is *absent* is how a
        serialiser that silently drops empty values gets caught.
        """
        records = [r.as_dict() for r in store.snapshot_requests()]
        if method:
            records = [r for r in records if r["method"] == method.upper()]
        if path:
            records = [r for r in records if r["path"] == path]
        return {"requests": records, "count": len(records)}

    @app.post(f"{ADMIN_PREFIX}reset", tags=["mock"])
    async def reset(requests_only: bool = False) -> dict[str, str]:
        """Clear state. ``?requests_only=true`` keeps stored objects."""
        if requests_only:
            store.clear_requests()
            return {"status": "requests cleared"}
        store.reset()
        if settings.seed:
            seed_store(store, settings)
        return {"status": "reset"}

    @app.get(f"{ADMIN_PREFIX}settings", tags=["mock"])
    async def current_settings() -> dict[str, Any]:
        """The active configuration, so a failing pipeline can show its quirks."""
        return {
            "user": settings.user,
            "seed": settings.seed,
            "frozen_clock": settings.frozen_clock,
            "requires_credentials": bool(settings.access_key or settings.secret_key),
            "quirks": {
                "on_omitted_description": settings.quirks.on_omitted_description.value,
                "on_omitted_filters": settings.quirks.on_omitted_filters.value,
                "lowercase_category_names": settings.quirks.lowercase_category_names,
                "reject_unknown_fields": settings.quirks.reject_unknown_fields,
            },
        }


#: Module-level app for ``uvicorn tenableio_mock.app:app``.
app = create_app()
