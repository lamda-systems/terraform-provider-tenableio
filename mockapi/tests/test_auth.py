"""The X-ApiKeys check, which applies uniformly to every Tenable.io path."""

from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from tenableio_mock.app import create_app
from tenableio_mock.config import Settings

from conftest import AUTH


def test_missing_header_is_rejected(client: TestClient) -> None:
    response = client.get("/tags/categories")
    assert response.status_code == 401
    assert response.json()["message"] == "Missing X-ApiKeys header."


@pytest.mark.parametrize(
    "header",
    [
        "garbage",
        "accessKey=only;",
        "secretKey=only;",
        "accessKey=;secretKey=;",
    ],
)
def test_malformed_header_is_rejected(client: TestClient, header: str) -> None:
    response = client.get("/tags/categories", headers={"X-ApiKeys": header})
    assert response.status_code == 401


def test_wellformed_header_is_accepted_when_no_credentials_are_pinned(
    client: TestClient,
) -> None:
    assert client.get("/tags/categories", headers=AUTH).status_code == 200


def test_pinned_credentials_are_enforced() -> None:
    app = create_app(Settings(access_key="right", secret_key="also-right"))
    client = TestClient(app)

    wrong = {"X-ApiKeys": "accessKey=wrong;secretKey=also-right;"}
    assert client.get("/tags/categories", headers=wrong).status_code == 401

    right = {"X-ApiKeys": "accessKey=right;secretKey=also-right;"}
    assert client.get("/tags/categories", headers=right).status_code == 200


def test_admin_endpoints_skip_authentication(client: TestClient) -> None:
    """A pipeline must be able to reach the mock's own controls unconditionally."""
    assert client.get("/__mock/health").status_code == 200
    assert client.get("/__mock/requests").status_code == 200
    assert client.post("/__mock/reset").status_code == 200


def test_error_envelope_is_tenable_shaped_not_fastapi_shaped(client: TestClient) -> None:
    """FastAPI's default 422 ``{"detail": [...]}`` would not match production."""
    response = client.post("/tags/categories", json={}, headers=AUTH)
    assert response.status_code == 400
    body = response.json()
    assert body["statusCode"] == 400
    assert body["error"] == "Bad Request"
    assert "detail" not in body
    assert "name" in body["message"]


def test_oversized_body_is_rejected(client: TestClient) -> None:
    response = client.post(
        "/tags/categories",
        json={"name": "x", "description": "y" * (1 << 21)},
        headers=AUTH,
    )
    assert response.status_code == 413


def test_documentation_endpoints_skip_authentication(client: TestClient) -> None:
    """A provider never calls these; requiring keys would only break the browser."""
    assert client.get("/openapi.json").status_code == 200
    assert client.get("/docs").status_code == 200


def test_documentation_traffic_is_not_recorded(client: TestClient) -> None:
    client.get("/openapi.json")
    assert client.get("/__mock/requests").json()["count"] == 0
