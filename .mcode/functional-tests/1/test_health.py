"""Functional tests for the GET /health/ endpoint (M1 foundation milestone).

The Flask origin and the Go target both expose `/health/` (with a literal
trailing slash). On success the endpoint must return:

    HTTP 200
    {"status": "healthy", "database": "healthy"}

Field order does not matter; the JSON object equality check below tolerates
arbitrary key ordering.

These tests are designed to run against either the origin (Flask) or the
target (Go) — the BASE_URL is read from the environment so the same script
can be shipped to either remote env.
"""

import os

import pytest
import requests


BASE_URL = os.environ.get("BASE_URL", "http://127.0.0.1:5000")
TIMEOUT = 10


@pytest.fixture(autouse=True)
def _service_reachable():
    """Confirm the service is actually up before each test runs."""
    try:
        resp = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
    except requests.RequestException as exc:
        pytest.fail(f"Service unreachable at {BASE_URL}: {exc}")
    assert resp.status_code in (200, 503), (
        f"Unexpected status from /health/: {resp.status_code}, body={resp.text!r}"
    )


class TestHealthEndpoint:
    """GET /health/ — the only endpoint delivered in M1."""

    def test_health_returns_200(self):
        resp = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
        assert resp.status_code == 200, (
            f"Expected 200, got {resp.status_code}: {resp.text!r}"
        )

    def test_health_returns_expected_json(self):
        resp = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
        assert resp.status_code == 200
        body = resp.json()
        assert body == {"status": "healthy", "database": "healthy"}, (
            f"Unexpected body: {body!r}"
        )

    def test_health_response_is_json(self):
        resp = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
        assert resp.status_code == 200
        ctype = resp.headers.get("Content-Type", "")
        assert "application/json" in ctype.lower(), (
            f"Expected JSON content type, got {ctype!r}"
        )

    def test_health_status_field(self):
        resp = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
        body = resp.json()
        assert body.get("status") == "healthy"

    def test_health_database_field(self):
        resp = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
        body = resp.json()
        assert body.get("database") == "healthy"


class TestHealthTrailingSlash:
    """Flask blueprint uses url_prefix=/health and route='/', so the
    canonical path includes the literal trailing slash."""

    def test_health_with_trailing_slash_is_canonical(self):
        resp = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT, allow_redirects=False)
        assert resp.status_code == 200, (
            f"GET /health/ must return 200 directly, got {resp.status_code}"
        )
