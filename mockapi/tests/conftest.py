"""Shared fixtures.

Every test drives the app through ``TestClient`` rather than poking the store
directly, so what is asserted is what a provider would actually see over HTTP.
"""

from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from tenableio_mock.app import create_app
from tenableio_mock.config import OnOmit, Quirks, ScanUpdateEcho, Settings

#: A syntactically valid header. The default Settings accepts any credentials,
#: so the values only matter in the tests that pin them.
AUTH = {"X-ApiKeys": "accessKey=access;secretKey=secret;"}


def make_client(**quirks) -> TestClient:
    """A client whose server has the given quirks enabled, all others off."""
    settings = Settings(quirks=Quirks(**quirks))
    return TestClient(create_app(settings))


@pytest.fixture
def client() -> TestClient:
    """The strict server: no quirks, seeded read-only data."""
    return make_client()


@pytest.fixture
def client_describing_scans() -> TestClient:
    """A server whose ``GET /scans/{id}`` reports a description. Not the default:
    live Tenable.io leaves the field out entirely."""
    return make_client(scan_details_description=True)


@pytest.fixture
def client_echoing_scan_updates() -> TestClient:
    """A server whose ``PUT /scans/{id}`` answers with the scan object."""
    return make_client(scan_update_echo=ScanUpdateEcho.OBJECT)


@pytest.fixture
def category(client: TestClient) -> dict:
    """A plain tag category to hang values off."""
    response = client.post("/tags/categories", json={"name": "Location"}, headers=AUTH)
    assert response.status_code == 200, response.text
    return response.json()


__all__ = [
    "AUTH",
    "make_client",
    "client",
    "client_describing_scans",
    "client_echoing_scan_updates",
    "category",
    "OnOmit",
    "ScanUpdateEcho",
]
