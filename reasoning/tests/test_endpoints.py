"""Tests for FastAPI endpoints."""

from datetime import datetime, timezone
from uuid import uuid4

import pytest
from fastapi.testclient import TestClient

from reasoning.app import app, _runs, _start_cache


@pytest.fixture(autouse=True)
def cleanup():
    """Clear global state between tests."""
    _runs.clear()
    _start_cache.clear()
    yield
    _runs.clear()
    _start_cache.clear()


@pytest.fixture
def client():
    return TestClient(app)


def test_healthz(client):
    resp = client.get("/healthz")
    assert resp.status_code == 200
    assert resp.json() == {"status": "ok"}


def test_cancel_not_found(client):
    resp = client.post("/reasoning/runs/nonexistent/cancel", json={
        "idempotency_key": "k1",
        "reason": "test",
    })
    assert resp.status_code == 404


def test_tool_results_not_found(client):
    resp = client.post("/reasoning/runs/nonexistent/tool-results", json={
        "idempotency_key": "k1",
        "tool_results": [],
    })
    assert resp.status_code == 404
