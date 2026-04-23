"""MCP facade using the official Python SDK. We wrap FastMCP so tests can
inspect and drive the tool handlers without spinning up the full transport.

Why a custom wrapper (TestableMCPServer) over plain FastMCP? The SDK's API
evolves, but the invariant we depend on is stable: FastMCP registers tools
as Python callables. We expose a minimal introspection surface
(``list_tools_sync``, ``call_tool_sync``) that the E2E test in Task 11
does NOT use — it goes through the protocol end-to-end.
"""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from mcp.server.fastmcp import FastMCP

from racd.core import ToolAdapter
from racd.schemas import DirRequest, GrepRequest, ReadRequest, TreeRequest


class ToolInfo:
    __slots__ = ("name", "handler")

    def __init__(self, name: str, handler: Callable[..., Any]) -> None:
        self.name = name
        self.handler = handler


class TestableMCPServer:
    def __init__(self, fastmcp: FastMCP, tools: dict[str, Callable[..., Any]]) -> None:
        self._fastmcp = fastmcp
        self._tools = tools

    @property
    def fastmcp(self) -> FastMCP:
        return self._fastmcp

    def list_tools_sync(self) -> list[ToolInfo]:
        return [ToolInfo(name, h) for name, h in self._tools.items()]

    async def call_tool_sync(self, name: str, args: dict[str, Any]) -> dict[str, Any]:
        handler = self._tools[name]
        result = await handler(**args)
        return result  # type: ignore[no-any-return]


def build_mcp_server(adapter: ToolAdapter) -> TestableMCPServer:
    mcp = FastMCP(name="racd")
    tools: dict[str, Callable[..., Any]] = {}

    @mcp.tool(
        name="read",
        description=(
            "Read a text file relative to the project root. Binary files are"
            " rejected. Returns up to 2000 lines or 256KB per call; paginate"
            " with offset/limit."
        ),
    )
    async def read_tool(
        path: str,
        offset: int | None = None,
        limit: int | None = None,
    ) -> dict[str, Any]:
        resp = await adapter.read(ReadRequest(path=path, offset=offset, limit=limit))
        return resp.model_dump()

    tools["read"] = read_tool

    @mcp.tool(
        name="grep",
        description=(
            "Search files under a path for a RE2 regex. Returns up to 200"
            " matches with optional context lines (0..5)."
        ),
    )
    async def grep_tool(
        pattern: str,
        path: str | None = None,
        include: str | None = None,
        context_lines: int | None = None,
        include_hidden: bool = False,
    ) -> dict[str, Any]:
        resp = await adapter.grep(
            GrepRequest(
                pattern=pattern,
                path=path,
                include=include,
                context_lines=context_lines,
                include_hidden=include_hidden,
            )
        )
        return resp.model_dump()

    tools["grep"] = grep_tool

    @mcp.tool(
        name="dir",
        description=(
            "List entries of a directory (non-recursive). Max 500 entries,"
            " sorted directories-first."
        ),
    )
    async def dir_tool(
        path: str | None = None,
        include_hidden: bool = False,
    ) -> dict[str, Any]:
        resp = await adapter.dir_(DirRequest(path=path, include_hidden=include_hidden))
        return resp.model_dump(exclude_none=True)

    tools["dir"] = dir_tool

    @mcp.tool(
        name="tree",
        description=(
            "Render a directory tree up to max_depth (default 3, max 10),"
            " capped at 5000 nodes."
        ),
    )
    async def tree_tool(
        path: str | None = None,
        max_depth: int | None = None,
        include_hidden: bool = False,
    ) -> dict[str, Any]:
        resp = await adapter.tree(
            TreeRequest(path=path, max_depth=max_depth, include_hidden=include_hidden)
        )
        return resp.model_dump()

    tools["tree"] = tree_tool

    return TestableMCPServer(mcp, tools)
