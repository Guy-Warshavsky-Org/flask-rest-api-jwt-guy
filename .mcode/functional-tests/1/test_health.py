"""Functional tests for the M1 /health/ endpoint of the Go service.

The Go service is a 1:1 functional port of the upstream Flask app, so this
suite is shared between origin and target. The base URL is read from
BASE_URL (preferred) or constructed from HOST/PORT, defaulting to
127.0.0.1:5050 to match the lifecycle harness's port for the Go service
(the upstream Flask service stays on 5000).

Covered cases (M1 only ships /health/):
  * test_health_trailing_slash_returns_200 - HAPPY_PATH
  * test_health_response_body_shape       - HAPPY_PATH
  * test_health_response_content_type     - HAPPY_PATH
  * test_health_no_trailing_slash_404     - NOT_FOUND
  * test_health_wrong_method_not_allowed  - INVALID_INPUT
"""
from __future__ import annotations

import json
import os

import pytest
import requests


def _base_url() -> str:
    if os.environ.get("BASE_URL"):
        return os.environ["BASE_URL"].rstrip("/")
    host = os.environ.get("HOST", "127.0.0.1")
    port = os.environ.get("PORT", "5050")
    return f"http://{host}:{port}"


BASE_URL = _base_url()
TIMEOUT = 10


@pytest.fixture(autouse=True)
def _service_reachable():
    """Sanity-check the service is up before each test runs."""
    try:
        resp = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
    except requests.exceptions.RequestException as exc:  # pragma: no cover
        pytest.fail(f"Service at {BASE_URL} not reachable: {exc!r}")
    assert resp.status_code in (200, 503), (
        f"Service replied with unexpected status {resp.status_code}: {resp.text!r}"
    )


class TestHealthEndpoint:
    """GET /health/ — the only endpoint shipped in M1."""

    def test_health_trailing_slash_returns_200(self):
        """Happy path: GET /health/ returns 200 when the DB is reachable."""
        resp = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
        assert resp.status_code == 200, (
            f"expected 200 OK, got {resp.status_code}: {resp.text!r}"
        )

    def test_health_response_body_shape(self):
        """Body must be JSON with status=healthy, database=healthy."""
        resp = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
        assert resp.status_code == 200
        body = resp.json()
        assert body == {"status": "healthy", "database": "healthy"}, (
            f"unexpected body: {body!r}"
        )

    def test_health_response_content_type(self):
        """Response must advertise itself as application/json."""
        resp = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
        ctype = resp.headers.get("Content-Type", "")
        assert "application/json" in ctype.lower(), (
            f"expected JSON content-type, got {ctype!r}"
        )
        # And the body must actually parse as JSON.
        json.loads(resp.text)


class TestHealthEndpointEdgeCases:
    """Behavior for paths/methods that should NOT match /health/."""

    def test_health_no_trailing_slash_404(self):
        """The Flask service registers /health/ literally; /health (no slash) MUST 404.

        We disable redirects so that frameworks which would redirect /health -> /health/
        (e.g. Flask's strict_slashes default) still surface a clear non-200 verdict.
        """
        resp = requests.get(
            f"{BASE_URL}/health", timeout=TIMEOUT, allow_redirects=False
        )
        # Either 404 (Gin's default for an unknown route) or 308/301 redirect to
        # /health/ (Flask's default) is acceptable - both prove /health (no slash)
        # is NOT the canonical endpoint. Critically, it must NOT be 200.
        assert resp.status_code != 200, (
            f"GET /health (no slash) should not be 200; got {resp.status_code}"
        )
        assert resp.status_code in (301, 308, 404), (
            f"unexpected status for /health (no slash): {resp.status_code}"
        )

    def test_health_wrong_method_not_allowed(self):
        """Non-GET methods on /health/ are not part of the contract."""
        resp = requests.post(f"{BASE_URL}/health/", timeout=TIMEOUT)
        # 404 (Gin default) or 405 (Flask default) - both indicate the method
        # is not part of the contract. Anything else is a bug.
        assert resp.status_code in (404, 405), (
            f"expected 404 or 405 for POST /health/, got {resp.status_code}"
        )
