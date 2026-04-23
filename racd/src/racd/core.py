"""Thin adapter that both facades (MCP + OpenAPI) share. It holds an
executor client and exposes one coroutine per tool. The adapter is the only
place that knows how to bridge Pydantic schemas with the executor.
"""

from __future__ import annotations

from typing import Protocol

from racd.schemas import (
    DirRequest,
    DirResponse,
    GrepRequest,
    GrepResponse,
    ReadRequest,
    ReadResponse,
    TreeRequest,
    TreeResponse,
)


class Executor(Protocol):
    async def read(self, req: ReadRequest) -> ReadResponse: ...
    async def grep(self, req: GrepRequest) -> GrepResponse: ...
    async def dir_(self, req: DirRequest) -> DirResponse: ...
    async def tree(self, req: TreeRequest) -> TreeResponse: ...


class ToolAdapter:
    def __init__(self, executor: Executor) -> None:
        self._ex = executor

    async def read(self, req: ReadRequest) -> ReadResponse:
        return await self._ex.read(req)

    async def grep(self, req: GrepRequest) -> GrepResponse:
        return await self._ex.grep(req)

    async def dir_(self, req: DirRequest) -> DirResponse:
        return await self._ex.dir_(req)

    async def tree(self, req: TreeRequest) -> TreeResponse:
        return await self._ex.tree(req)
