"""Tests for the executor HTTP client. We inject httpx.MockTransport so we
exercise the full request/response flow without a real racx."""

from __future__ import annotations

import json
from typing import Any

import httpx
import pytest

from racd.executor_client import ExecutorClient, ExecutorError
from racd.schemas import (
    DirRequest,
    DirResponse,
    ErrorCode,
    GrepRequest,
    GrepResponse,
    ReadRange,
    ReadRequest,
    ReadResponse,
    TreeRequest,
    TreeResponse,
)


def _transport(handler: httpx.MockTransport) -> ExecutorClient:
    return ExecutorClient(base_url="http://racx.test", transport=handler)


async def test_ping_ok() -> None:
    def handler(req: httpx.Request) -> httpx.Response:
        assert req.method == "GET"
        assert req.url.path == "/ping"
        return httpx.Response(200, json={"ok": True})

    client = _transport(httpx.MockTransport(handler))
    assert await client.ping() is True
    await client.aclose()


async def test_read_happy_path() -> None:
    captured: dict[str, Any] = {}

    def handler(req: httpx.Request) -> httpx.Response:
        captured["path"] = req.url.path
        captured["body"] = json.loads(req.content)
        return httpx.Response(
            200,
            json={
                "content": "hello",
                "total_lines": 1,
                "truncated": False,
                "returned_range": {"start": 1, "end": 1},
            },
        )

    client = _transport(httpx.MockTransport(handler))
    resp = await client.read(ReadRequest(path="a.txt"))
    assert isinstance(resp, ReadResponse)
    assert resp.content == "hello"
    assert resp.returned_range == ReadRange(start=1, end=1)
    assert captured["path"] == "/read"
    assert captured["body"] == {"path": "a.txt"}
    await client.aclose()


async def test_denied_by_policy_surfaces_as_executor_error() -> None:
    def handler(req: httpx.Request) -> httpx.Response:
        return httpx.Response(
            403,
            json={
                "error": {
                    "code": "denied_by_policy",
                    "message": "outside root",
                }
            },
        )

    client = _transport(httpx.MockTransport(handler))
    with pytest.raises(ExecutorError) as exc_info:
        await client.read(ReadRequest(path="../etc/passwd"))
    assert exc_info.value.code is ErrorCode.DENIED_BY_POLICY
    assert exc_info.value.message == "outside root"
    await client.aclose()


async def test_timeout_becomes_executor_timeout() -> None:
    def handler(req: httpx.Request) -> httpx.Response:
        raise httpx.TimeoutException("slow", request=req)

    client = _transport(httpx.MockTransport(handler))
    with pytest.raises(ExecutorError) as exc_info:
        await client.read(ReadRequest(path="a.txt"))
    assert exc_info.value.code is ErrorCode.EXECUTOR_TIMEOUT
    await client.aclose()


async def test_connection_error_becomes_executor_unavailable() -> None:
    def handler(req: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("refused", request=req)

    client = _transport(httpx.MockTransport(handler))
    with pytest.raises(ExecutorError) as exc_info:
        await client.read(ReadRequest(path="a.txt"))
    assert exc_info.value.code is ErrorCode.EXECUTOR_UNAVAILABLE
    await client.aclose()


async def test_grep_dir_tree_routes() -> None:
    routes: list[str] = []

    def handler(req: httpx.Request) -> httpx.Response:
        routes.append(req.url.path)
        if req.url.path == "/grep":
            return httpx.Response(200, json={"matches": [], "total": 0, "truncated": False})
        if req.url.path == "/dir":
            return httpx.Response(200, json={"entries": [], "total": 0, "truncated": False})
        if req.url.path == "/tree":
            return httpx.Response(
                200,
                json={"rendered": "root\n", "total_nodes": 1, "truncated": False},
            )
        raise AssertionError(f"unexpected path: {req.url.path}")

    client = _transport(httpx.MockTransport(handler))
    assert isinstance(await client.grep(GrepRequest(pattern="x")), GrepResponse)
    assert isinstance(await client.dir_(DirRequest()), DirResponse)
    assert isinstance(await client.tree(TreeRequest()), TreeResponse)
    assert routes == ["/grep", "/dir", "/tree"]
    await client.aclose()


async def test_malformed_error_body_still_raises_executor_error() -> None:
    def handler(req: httpx.Request) -> httpx.Response:
        return httpx.Response(500, text="not-json")

    client = _transport(httpx.MockTransport(handler))
    with pytest.raises(ExecutorError) as exc_info:
        await client.read(ReadRequest(path="a.txt"))
    assert exc_info.value.code is ErrorCode.IO_ERROR
    await client.aclose()
