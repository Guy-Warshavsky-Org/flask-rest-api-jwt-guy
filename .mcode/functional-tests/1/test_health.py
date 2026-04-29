"""Functional tests for GET /health/ on the Flask origin service.

These tests run inside the model-daemon origin environment (Flask on PORT,
default 5000). They establish the behavioral baseline that the Go target
service must match for the `health` entity.
"""
from __future__ import annotations

import json
import os

import pytest
import requests

PORT = int(os.environ.get("PORT", "5000"))
BASE_URL = f"http://127.0.0.1:{PORT}"
TIMEOUT = 5


@pytest.fixture(scope="module", autouse=True)
def _wait_for_app() -> None:
    """Smoke-check that the app is reachable before running the suite."""
    last_exc: Exception | None = None
    for _ in range(20):
        try:
            r = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
            if r.status_code in (200, 503):
                return
        except Exception as e:  # noqa: BLE001 - retry until ready
            last_exc = e
        import time

        time.sleep(0.5)
    raise RuntimeError(f"App never became reachable at {BASE_URL}: {last_exc}")


class TestHealth:
    """Tests for the /health/ endpoint."""

    def test_health_get_returns_200(self) -> None:
        """HAPPY_PATH: GET /health/ returns 200 with healthy envelope."""
        r = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
        assert r.status_code == 200, r.text
        body = r.json()
        assert body == {"status": "healthy", "database": "healthy"}, body
        print("ACTUAL_BODY:", json.dumps(body, sort_keys=True))
        print("ACTUAL_STATUS:", r.status_code)

    def test_health_content_type_is_json(self) -> None:
        """HAPPY_PATH: response Content-Type is application/json."""
        r = requests.get(f"{BASE_URL}/health/", timeout=TIMEOUT)
        assert r.status_code == 200, r.text
        ctype = r.headers.get("Content-Type", "")
        assert "application/json" in ctype.lower(), ctype
        print("ACTUAL_CTYPE:", ctype)

    def test_health_without_trailing_slash_is_not_200(self) -> None:
        """BOUNDARY: /health (no trailing slash) is not the canonical route."""
        r = requests.get(
            f"{BASE_URL}/health", timeout=TIMEOUT, allow_redirects=False
        )
        assert r.status_code != 200, (
            f"GET /health (no trailing slash) unexpectedly returned 200: {r.text}"
        )
        print("ACTUAL_STATUS_NO_SLASH:", r.status_code)

    def test_health_post_not_allowed(self) -> None:
        """INVALID_INPUT: POST /health/ is not a registered method."""
        r = requests.post(f"{BASE_URL}/health/", timeout=TIMEOUT)
        assert r.status_code != 200, r.text
        assert 400 <= r.status_code < 500, r.status_code
        print("ACTUAL_STATUS_POST:", r.status_code)
