"""Compose the full ASGI application: FastAPI root + OpenAPI facade at
``/v1`` + MCP streamable HTTP app mounted at ``/mcp``. Also exposes an
unauthenticated ``/healthz`` endpoint.
"""

from __future__ import annotations

from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastapi import FastAPI

from racd.auth import install_handler as install_auth_handler
from racd.config import Config
from racd.core import ToolAdapter
from racd.executor_client import ExecutorClient
from racd.mcp_facade import build_mcp_server
from racd.openapi_facade import build_router, install_exception_handler


def build_app(config: Config, executor: ToolAdapter | None = None) -> FastAPI:
    if executor is None:
        client = ExecutorClient(base_url=f"http://{config.executor.host}:{config.executor.port}")
        executor = ToolAdapter(client)
        owns_client: ExecutorClient | None = client
    else:
        owns_client = None

    @asynccontextmanager
    async def lifespan(_: FastAPI) -> AsyncIterator[None]:
        try:
            yield
        finally:
            if owns_client is not None:
                await owns_client.aclose()

    app = FastAPI(title="racd", version="0.1.0", lifespan=lifespan)
    install_exception_handler(app)
    install_auth_handler(app)
    app.include_router(build_router(executor, config), prefix="/v1")

    mcp = build_mcp_server(executor)
    app.mount("/mcp", mcp.fastmcp.streamable_http_app())

    @app.get("/healthz")
    async def healthz() -> dict[str, bool]:
        return {"ok": True}

    return app
