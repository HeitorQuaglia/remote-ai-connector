"""Tests for the OpenAPI facade."""

from __future__ import annotations

from dataclasses import dataclass

from fastapi import FastAPI
from fastapi.testclient import TestClient

from racd.auth import install_handler as install_auth_handler
from racd.config import ApiKey, Config, ExecutorConfig
from racd.core import ToolAdapter
from racd.executor_client import ExecutorError
from racd.openapi_facade import build_router, install_exception_handler
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


@dataclass
class StubExecutor:
    read_return: ReadResponse | Exception | None = None
    grep_return: GrepResponse | Exception | None = None
    dir_return: DirResponse | Exception | None = None
    tree_return: TreeResponse | Exception | None = None

    async def read(self, _: ReadRequest) -> ReadResponse:
        if isinstance(self.read_return, Exception):
            raise self.read_return
        assert self.read_return is not None
        return self.read_return

    async def grep(self, _: GrepRequest) -> GrepResponse:
        if isinstance(self.grep_return, Exception):
            raise self.grep_return
        assert self.grep_return is not None
        return self.grep_return

    async def dir_(self, _: DirRequest) -> DirResponse:
        if isinstance(self.dir_return, Exception):
            raise self.dir_return
        assert self.dir_return is not None
        return self.dir_return

    async def tree(self, _: TreeRequest) -> TreeResponse:
        if isinstance(self.tree_return, Exception):
            raise self.tree_return
        assert self.tree_return is not None
        return self.tree_return


def _cfg() -> Config:
    return Config(
        listen="127.0.0.1:0",
        api_keys=[ApiKey(name="claude-ai", key="secret")],
        executor=ExecutorConfig(host="127.0.0.1", port=7001),
    )


def _app(adapter: ToolAdapter, cfg: Config) -> FastAPI:
    app = FastAPI()
    install_exception_handler(app)
    install_auth_handler(app)
    app.include_router(build_router(adapter, cfg), prefix="/v1")

    @app.get("/healthz")
    def healthz() -> dict[str, bool]:
        return {"ok": True}

    return app


def test_healthz_unauthenticated() -> None:
    app = _app(
        ToolAdapter(StubExecutor(read_return=None)),  # type: ignore[arg-type]
        _cfg(),
    )
    r = TestClient(app).get("/healthz")
    assert r.status_code == 200
    assert r.json() == {"ok": True}


def test_read_route_requires_api_key() -> None:
    app = _app(
        ToolAdapter(StubExecutor()),  # type: ignore[arg-type]
        _cfg(),
    )
    r = TestClient(app).post("/v1/read", json={"path": "a.txt"})
    assert r.status_code == 401


def test_read_route_success() -> None:
    executor = StubExecutor(
        read_return=ReadResponse(
            content="hi",
            total_lines=1,
            truncated=False,
            returned_range=ReadRange(start=1, end=1),
        )
    )
    app = _app(ToolAdapter(executor), _cfg())  # type: ignore[arg-type]
    r = TestClient(app).post(
        "/v1/read",
        json={"path": "a.txt"},
        headers={"Authorization": "Bearer secret"},
    )
    assert r.status_code == 200
    assert r.json()["content"] == "hi"


def test_read_route_denied_maps_to_403() -> None:
    executor = StubExecutor(read_return=ExecutorError(ErrorCode.DENIED_BY_POLICY, "no"))
    app = _app(ToolAdapter(executor), _cfg())  # type: ignore[arg-type]
    r = TestClient(app).post(
        "/v1/read",
        json={"path": "../etc/passwd"},
        headers={"Authorization": "Bearer secret"},
    )
    assert r.status_code == 403
    assert r.json()["error"]["code"] == "denied_by_policy"


def test_read_route_not_found_maps_to_404() -> None:
    executor = StubExecutor(read_return=ExecutorError(ErrorCode.NOT_FOUND, "nope"))
    app = _app(ToolAdapter(executor), _cfg())  # type: ignore[arg-type]
    r = TestClient(app).post(
        "/v1/read",
        json={"path": "a.txt"},
        headers={"Authorization": "Bearer secret"},
    )
    assert r.status_code == 404


def test_executor_unavailable_maps_to_502() -> None:
    executor = StubExecutor(
        read_return=ExecutorError(ErrorCode.EXECUTOR_UNAVAILABLE, "down")
    )
    app = _app(ToolAdapter(executor), _cfg())  # type: ignore[arg-type]
    r = TestClient(app).post(
        "/v1/read",
        json={"path": "a.txt"},
        headers={"Authorization": "Bearer secret"},
    )
    assert r.status_code == 502
    assert r.json()["error"]["code"] == "executor_unavailable"


def test_openapi_spec_includes_tools() -> None:
    app = _app(
        ToolAdapter(StubExecutor()),  # type: ignore[arg-type]
        _cfg(),
    )
    spec = TestClient(app).get("/openapi.json").json()
    assert "/v1/read" in spec["paths"]
    assert "/v1/grep" in spec["paths"]
    assert "/v1/dir" in spec["paths"]
    assert "/v1/tree" in spec["paths"]


def test_grep_route_success() -> None:
    executor = StubExecutor(
        grep_return=GrepResponse(matches=[], total=0, truncated=False)
    )
    app = _app(ToolAdapter(executor), _cfg())  # type: ignore[arg-type]
    r = TestClient(app).post(
        "/v1/grep",
        json={"pattern": "TODO"},
        headers={"Authorization": "Bearer secret"},
    )
    assert r.status_code == 200


def test_dir_route_success() -> None:
    executor = StubExecutor(dir_return=DirResponse(entries=[], total=0, truncated=False))
    app = _app(ToolAdapter(executor), _cfg())  # type: ignore[arg-type]
    r = TestClient(app).post(
        "/v1/dir",
        json={},
        headers={"Authorization": "Bearer secret"},
    )
    assert r.status_code == 200


def test_tree_route_success() -> None:
    executor = StubExecutor(
        tree_return=TreeResponse(rendered="root\n", total_nodes=1, truncated=False)
    )
    app = _app(ToolAdapter(executor), _cfg())  # type: ignore[arg-type]
    r = TestClient(app).post(
        "/v1/tree",
        json={},
        headers={"Authorization": "Bearer secret"},
    )
    assert r.status_code == 200
