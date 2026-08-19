"""Tenable-shaped error responses.

FastAPI's defaults are wrong for this job in two ways: validation failures come
back as ``422`` with a ``{"detail": [...]}`` body, and ``HTTPException`` renders
as ``{"detail": "..."}``. Tenable.io returns neither. Since the whole point of
the mock is that a provider which passes against it also passes against
production, the handlers here replace both.

The envelope itself is an approximation. Tenable.io is not consistent across
endpoints and the public docs quote error *messages* far more often than they
quote the JSON around them. What is faithful, and what the provider can
actually observe, is the status code and the message text --
``internal/client`` turns any non-2xx into an ``APIError`` carrying the raw
body. Do not assert on the envelope from provider code.
"""

from __future__ import annotations

from typing import Any

from fastapi import FastAPI, Request, status
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException as StarletteHTTPException

_STATUS_TEXT = {
    status.HTTP_400_BAD_REQUEST: "Bad Request",
    status.HTTP_401_UNAUTHORIZED: "Unauthorized",
    status.HTTP_403_FORBIDDEN: "Forbidden",
    status.HTTP_404_NOT_FOUND: "Not Found",
    status.HTTP_409_CONFLICT: "Conflict",
    413: "Payload Too Large",
    status.HTTP_429_TOO_MANY_REQUESTS: "Too Many Requests",
    status.HTTP_500_INTERNAL_SERVER_ERROR: "Internal Server Error",
}


def error_body(status_code: int, message: str, code: str = "") -> dict[str, Any]:
    body: dict[str, Any] = {
        "statusCode": status_code,
        "error": _STATUS_TEXT.get(status_code, "Error"),
        "message": message,
    }
    if code:
        # The tags endpoints document short machine-readable codes
        # ("duplicate", "not_found"); other endpoints document none.
        body["code"] = code
    return body


class TenableError(Exception):
    """Raise from a handler to produce a Tenable-shaped error response."""

    def __init__(self, status_code: int, message: str, code: str = "") -> None:
        super().__init__(message)
        self.status_code = status_code
        self.message = message
        self.code = code


def bad_request(message: str, code: str = "") -> TenableError:
    return TenableError(status.HTTP_400_BAD_REQUEST, message, code)


def not_found(message: str, code: str = "not_found") -> TenableError:
    return TenableError(status.HTTP_404_NOT_FOUND, message, code)


def json_error(status_code: int, message: str, code: str = "") -> JSONResponse:
    return JSONResponse(
        status_code=status_code, content=error_body(status_code, message, code)
    )


def install_handlers(app: FastAPI) -> None:
    @app.exception_handler(TenableError)
    async def _tenable(_: Request, exc: TenableError) -> JSONResponse:
        return json_error(exc.status_code, exc.message, exc.code)

    @app.exception_handler(RequestValidationError)
    async def _validation(_: Request, exc: RequestValidationError) -> JSONResponse:
        # Collapse pydantic's structured report into the single human sentence
        # Tenable.io would have returned, and downgrade 422 to 400.
        return json_error(status.HTTP_400_BAD_REQUEST, _describe(exc))

    @app.exception_handler(StarletteHTTPException)
    async def _http(_: Request, exc: StarletteHTTPException) -> JSONResponse:
        detail = exc.detail if isinstance(exc.detail, str) else str(exc.detail)
        return json_error(exc.status_code, detail)


def _describe(exc: RequestValidationError) -> str:
    parts: list[str] = []
    for err in exc.errors():
        # loc is ("body", "field", ...); drop the leading source marker.
        loc = [str(p) for p in err.get("loc", ()) if p not in ("body", "query", "path")]
        where = ".".join(loc) if loc else "request"
        parts.append(f"{where}: {err.get('msg', 'invalid')}")
    return "; ".join(parts) or "Malformed request body."
