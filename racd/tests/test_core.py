"""Tests for the core tool adapter. Uses a fake executor client that records
calls and returns canned responses."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

import pytest

from racd.core import ToolAdapter
from racd.executor_client import ExecutorError
from racd.schemas import (
    DirEntry,
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
class FakeExecutor:
    read_return: ReadResponse | Exception | None = None
    grep_return: GrepResponse | Exception | None = None
    dir_return: DirResponse | Exception | None = None
    tree_return: TreeResponse | Exception | None = None
    calls: list[tuple[str, Any]] = field(default_factory=list)

    async def read(self, req: ReadRequest) -> ReadResponse:
        self.calls.append(("read", req))
        return self._yield(self.read_return)

    async def grep(self, req: GrepRequest) -> GrepResponse:
        self.calls.append(("grep", req))
        return self._yield(self.grep_return)

    async def dir_(self, req: DirRequest) -> DirResponse:
        self.calls.append(("dir_", req))
        return self._yield(self.dir_return)

    async def tree(self, req: TreeRequest) -> TreeResponse:
        self.calls.append(("tree", req))
        return self._yield(self.tree_return)

    def _yield(self, value: Any) -> Any:
        if isinstance(value, Exception):
            raise value
        return value


async def test_read_passes_request_through() -> None:
    fake = FakeExecutor(
        read_return=ReadResponse(
            content="hi",
            total_lines=1,
            truncated=False,
            returned_range=ReadRange(start=1, end=1),
        )
    )
    adapter = ToolAdapter(fake)  # type: ignore[arg-type]
    resp = await adapter.read(ReadRequest(path="a.txt"))
    assert resp.content == "hi"
    assert fake.calls[0][0] == "read"


async def test_read_propagates_executor_error() -> None:
    fake = FakeExecutor(read_return=ExecutorError(ErrorCode.NOT_FOUND, "nope"))
    adapter = ToolAdapter(fake)  # type: ignore[arg-type]
    with pytest.raises(ExecutorError) as exc_info:
        await adapter.read(ReadRequest(path="a.txt"))
    assert exc_info.value.code is ErrorCode.NOT_FOUND


async def test_grep_dir_tree_happy() -> None:
    fake = FakeExecutor(
        grep_return=GrepResponse(matches=[], total=0, truncated=False),
        dir_return=DirResponse(
            entries=[DirEntry(name="a", type="file", size=1)],
            total=1,
            truncated=False,
        ),
        tree_return=TreeResponse(rendered="a\n", total_nodes=1, truncated=False),
    )
    adapter = ToolAdapter(fake)  # type: ignore[arg-type]

    gr = await adapter.grep(GrepRequest(pattern="x"))
    assert gr.total == 0
    dr = await adapter.dir_(DirRequest())
    assert dr.entries[0].name == "a"
    tr = await adapter.tree(TreeRequest())
    assert tr.total_nodes == 1
