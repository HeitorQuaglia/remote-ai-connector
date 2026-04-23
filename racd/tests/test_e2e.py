"""End-to-end test: racd OpenAPI facade talking to a stubbed racx."""

from __future__ import annotations

import json
from collections.abc import Iterator

import httpx
import pytest
from fastapi.testclient import TestClient

from racd.app import build_app
from racd.config import ApiKey, Config, ExecutorConfig
from racd.core import ToolAdapter
from racd.executor_client import ExecutorClient


def _build_stub_racx_transport() -> httpx.BaseTransport:
    """Build a mock transport that simulates a stubbed racx server."""

    def handler(request: httpx.Request) -> httpx.Response:
        if request.method == "GET" and request.url.path == "/ping":
            return httpx.Response(200, json={"ok": True})

        if request.method == "POST" and request.url.path == "/read":
            body = json.loads(request.content)
            if body.get("path") == "secrets/.env":
                return httpx.Response(
                    403,
                    json={"error": {"code": "denied_by_policy", "message": "denylist"}},
                )
            return httpx.Response(
                200,
                json={
                    "content": f"echo:{body.get('path')}",
                    "total_lines": 1,
                    "truncated": False,
                    "returned_range": {"start": 1, "end": 1},
                },
            )

        if request.method == "POST" and request.url.path == "/grep":
            body = json.loads(request.content)
            return httpx.Response(
                200,
                json={
                    "matches": [
                        {
                            "file": "a.py",
                            "line": 1,
                            "column": 1,
                            "text": body["pattern"],
                        }
                    ],
                    "total": 1,
                    "truncated": False,
                },
            )

        if request.method == "POST" and request.url.path == "/dir":
            return httpx.Response(
                200,
                json={
                    "entries": [{"name": "README.md", "type": "file", "size": 42}],
                    "total": 1,
                    "truncated": False,
                },
            )

        if request.method == "POST" and request.url.path == "/tree":
            return httpx.Response(
                200,
                json={
                    "rendered": "root\n└── a\n",
                    "total_nodes": 2,
                    "truncated": False,
                },
            )

        return httpx.Response(404, json={"error": {"code": "not_found", "message": "unknown endpoint"}})

    return httpx.MockTransport(handler)


@pytest.fixture
def e2e_client() -> Iterator[TestClient]:
    transport = _build_stub_racx_transport()
    executor_client = ExecutorClient(base_url="http://racx.test", transport=transport)

    config = Config(
        listen="127.0.0.1:0",
        api_keys=[ApiKey(name="test", key="secret")],
        executor=ExecutorConfig(host="127.0.0.1", port=7001),
    )
    app = build_app(config, executor=ToolAdapter(executor_client))
    client = TestClient(app)
    try:
        yield client
    finally:
        client.close()


def test_e2e_read_happy(e2e_client: TestClient) -> None:
    r = e2e_client.post(
        "/v1/read",
        json={"path": "a.txt"},
        headers={"Authorization": "Bearer secret"},
    )
    assert r.status_code == 200
    assert r.json()["content"] == "echo:a.txt"


def test_e2e_read_denied(e2e_client: TestClient) -> None:
    r = e2e_client.post(
        "/v1/read",
        json={"path": "secrets/.env"},
        headers={"Authorization": "Bearer secret"},
    )
    assert r.status_code == 403
    assert r.json()["error"]["code"] == "denied_by_policy"


def test_e2e_grep(e2e_client: TestClient) -> None:
    r = e2e_client.post(
        "/v1/grep",
        json={"pattern": "TODO"},
        headers={"Authorization": "Bearer secret"},
    )
    assert r.status_code == 200
    assert r.json()["matches"][0]["text"] == "TODO"


def test_e2e_tree(e2e_client: TestClient) -> None:
    r = e2e_client.post(
        "/v1/tree",
        json={},
        headers={"Authorization": "Bearer secret"},
    )
    assert r.status_code == 200
    assert "└──" in r.json()["rendered"]


def test_e2e_dir(e2e_client: TestClient) -> None:
    r = e2e_client.post(
        "/v1/dir",
        json={},
        headers={"Authorization": "Bearer secret"},
    )
    assert r.status_code == 200
    assert r.json()["entries"][0]["name"] == "README.md"


def test_e2e_healthz_unauthenticated(e2e_client: TestClient) -> None:
    r = e2e_client.get("/healthz")
    assert r.status_code == 200
