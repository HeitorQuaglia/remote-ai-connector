"""Tests for the MCP facade. We call the registered tool callables directly
(via FastMCP's _tool_manager) to avoid depending on protocol internals."""

from __future__ import annotations

from dataclasses import dataclass

import pytest

from racd.core import ToolAdapter
from racd.executor_client import ExecutorError
from racd.mcp_facade import build_mcp_server
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


def test_all_four_tools_registered() -> None:
    mcp = build_mcp_server(ToolAdapter(StubExecutor()))  # type: ignore[arg-type]
    names = {t.name for t in mcp.list_tools_sync()}
    assert names == {"read", "grep", "dir", "tree"}


async def test_read_tool_returns_response_dict() -> None:
    executor = StubExecutor(
        read_return=ReadResponse(
            content="hi",
            total_lines=1,
            truncated=False,
            returned_range=ReadRange(start=1, end=1),
        )
    )
    mcp = build_mcp_server(ToolAdapter(executor))  # type: ignore[arg-type]
    result = await mcp.call_tool_sync("read", {"path": "a.txt"})
    assert result["content"] == "hi"
    assert result["total_lines"] == 1


async def test_read_tool_denied_raises_executor_error() -> None:
    executor = StubExecutor(read_return=ExecutorError(ErrorCode.DENIED_BY_POLICY, "x"))
    mcp = build_mcp_server(ToolAdapter(executor))  # type: ignore[arg-type]
    with pytest.raises(ExecutorError):
        await mcp.call_tool_sync("read", {"path": "../etc/passwd"})


async def test_grep_dir_tree_happy() -> None:
    executor = StubExecutor(
        grep_return=GrepResponse(matches=[], total=0, truncated=False),
        dir_return=DirResponse(entries=[], total=0, truncated=False),
        tree_return=TreeResponse(rendered="root\n", total_nodes=1, truncated=False),
    )
    mcp = build_mcp_server(ToolAdapter(executor))  # type: ignore[arg-type]
    assert (await mcp.call_tool_sync("grep", {"pattern": "x"}))["total"] == 0
    assert (await mcp.call_tool_sync("dir", {}))["total"] == 0
    assert (await mcp.call_tool_sync("tree", {}))["total_nodes"] == 1
