"""Pydantic schemas for the racd server. These mirror the JSON shapes of the
racx Go types under internal/tools/types.go. Any change here must be kept in
sync with the Go side.

The envelope shape for errors is ``{"error": {"code": ..., "message": ...}}``,
following the spec decision to make both facades emit the same JSON.
"""

from __future__ import annotations

from enum import StrEnum
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field


class ErrorCode(StrEnum):
    NOT_FOUND = "not_found"
    IS_DIRECTORY = "is_directory"
    BINARY_FILE = "binary_file"
    DENIED_BY_POLICY = "denied_by_policy"
    FILE_TOO_LARGE = "file_too_large"
    IO_ERROR = "io_error"
    INVALID_REGEX = "invalid_regex"
    INVALID_ARGUMENT = "invalid_argument"
    EXECUTOR_UNAVAILABLE = "executor_unavailable"
    EXECUTOR_TIMEOUT = "executor_timeout"


class Error(BaseModel):
    model_config = ConfigDict(extra="forbid")

    code: ErrorCode
    message: str
    details: dict[str, Any] | None = None


class ErrorEnvelope(BaseModel):
    model_config = ConfigDict(extra="forbid")

    error: Error


# ----- Read -----


class ReadRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    path: str = Field(..., description="Path relative to project root.")
    offset: int | None = Field(None, description="1-based starting line (default 1).")
    limit: int | None = Field(None, description="Maximum lines to return (default 2000).")


class ReadRange(BaseModel):
    model_config = ConfigDict(extra="forbid")

    start: int
    end: int


class ReadResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    content: str
    total_lines: int
    truncated: bool
    returned_range: ReadRange


# ----- Grep -----


class GrepRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    pattern: str
    path: str | None = None
    include: str | None = None
    context_lines: int | None = Field(None, ge=0, le=5)
    include_hidden: bool = False


class GrepMatch(BaseModel):
    model_config = ConfigDict(extra="forbid")

    file: str
    line: int
    column: int
    text: str
    before: list[str] = Field(default_factory=list)
    after: list[str] = Field(default_factory=list)


class GrepResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    matches: list[GrepMatch]
    total: int
    truncated: bool


# ----- Dir -----


DirEntryType = Literal["file", "dir", "symlink"]


class DirEntry(BaseModel):
    model_config = ConfigDict(extra="forbid")

    name: str
    type: DirEntryType
    size: int | None = None
    symlink_target: str | None = None


class DirRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    path: str | None = None
    include_hidden: bool = False


class DirResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    entries: list[DirEntry]
    total: int
    truncated: bool


# ----- Tree -----


class TreeRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    path: str | None = None
    max_depth: int | None = Field(None, ge=1, le=10)
    include_hidden: bool = False


class TreeResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    rendered: str
    total_nodes: int
    truncated: bool
