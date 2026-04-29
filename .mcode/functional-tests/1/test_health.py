"""M1 functional tests for the /health/ endpoint — origin (Flask) probe.

Mirror of the target's pytest functional tests so the same shape of responses
(status code, JSON keys) is verified against both implementations during the
two-pass functional verification workflow.

The bare-/health behavior is INTENTIONALLY divergent between origin and
target: Werkzeug's default `strict_slashes=True` makes Flask redirect bare
paths to their trailing-slash form with HTTP 308, while the Go target
disables `RedirectTrailingSlash` and returns 404 instead. The Go service's
manifest entry for that probe is therefore `target_only`; the corresponding
test class below documents the origin's actual behavior so this file passes
when run against the running Flask service.

Tests rely on `BASE_URL` (env var) or fall back to ``http://localhost:5000``
for the origin Flask service.
"""

from __future__ import annotations

import os

import pytest
import requests

BASE_URL = os.environ.get("BASE_URL", "http://localhost:5000").rstrip("/")
HTTP_TIMEOUT = 5


@pytest.fixture(autouse=True)
def health_check():
    """Confirm the app is reachable before running any test in this module."""
    resp = requests.get(
        f"{BASE_URL}/health/",
        timeout=HTTP_TIMEOUT,
        allow_redirects=False,
    )
    assert resp.status_code == 200, (
        f"App not reachable at {BASE_URL}/health/ (status={resp.status_code})"
    )


class TestHealthHappyPath:
    """GET /health/ — must return 200 with the canonical JSON shape."""

    def test_health_returns_200(self):
        resp = requests.get(
            f"{BASE_URL}/health/",
            timeout=HTTP_TIMEOUT,
            allow_redirects=False,
        )
        assert resp.status_code == 200, (
            f"expected 200 from /health/, got {resp.status_code}: {resp.text}"
        )

    def test_health_returns_expected_json_shape(self):
        resp = requests.get(
            f"{BASE_URL}/health/",
            timeout=HTTP_TIMEOUT,
            allow_redirects=False,
        )
        body = resp.json()
        assert body.get("status") == "healthy", (
            f'expected status="healthy", got body={body}'
        )
        assert body.get("database") == "healthy", (
            f'expected database="healthy", got body={body}'
        )

    def test_health_content_type_is_json(self):
        resp = requests.get(
            f"{BASE_URL}/health/",
            timeout=HTTP_TIMEOUT,
            allow_redirects=False,
        )
        ctype = resp.headers.get("Content-Type", "")
        assert "application/json" in ctype.lower(), (
            f"expected JSON content-type, got: {ctype}"
        )


class TestUnrelatedPath:
    """An unrelated path must 404 cleanly, not 5xx and not redirect."""

    def test_missing_path_returns_404(self):
        resp = requests.get(
            f"{BASE_URL}/missing",
            timeout=HTTP_TIMEOUT,
            allow_redirects=False,
        )
        assert resp.status_code == 404, (
            f"expected 404 on /missing, got {resp.status_code}: {resp.text!r}"
        )
