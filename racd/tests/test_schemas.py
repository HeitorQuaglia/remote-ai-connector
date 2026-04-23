"""Schema tests verify JSON round-trip parity with racx Go types."""

import json

from racd.schemas import (
    DirEntry,
    DirRequest,
    DirResponse,
    Error,
    ErrorCode,
    ErrorEnvelope,
    GrepMatch,
    GrepRequest,
    GrepResponse,
    ReadRange,
    ReadRequest,
    ReadResponse,
    TreeRequest,
    TreeResponse,
)


def test_error_code_values() -> None:
    assert ErrorCode.NOT_FOUND.value == "not_found"
    assert ErrorCode.DENIED_BY_POLICY.value == "denied_by_policy"
    assert ErrorCode.BINARY_FILE.value == "binary_file"


def test_error_envelope_roundtrip() -> None:
    env = ErrorEnvelope(error=Error(code=ErrorCode.DENIED_BY_POLICY, message="blocked"))
    j = env.model_dump_json()
    back = ErrorEnvelope.model_validate_json(j)
    assert back.error.code is ErrorCode.DENIED_BY_POLICY
    assert back.error.message == "blocked"


def test_read_request_optional_fields_omitted() -> None:
    req = ReadRequest(path="src/app.py")
    dumped = json.loads(req.model_dump_json(exclude_none=True))
    assert dumped == {"path": "src/app.py"}


def test_read_response_roundtrip_with_range() -> None:
    resp = ReadResponse(
        content="line1\nline2\n",
        total_lines=2,
        truncated=False,
        returned_range=ReadRange(start=1, end=2),
    )
    back = ReadResponse.model_validate_json(resp.model_dump_json())
    assert back.returned_range.start == 1
    assert back.returned_range.end == 2


def test_grep_request_accepts_context_lines() -> None:
    req = GrepRequest(pattern="TODO", context_lines=2)
    assert req.context_lines == 2


def test_grep_response_with_matches() -> None:
    resp = GrepResponse(
        matches=[GrepMatch(file="a.py", line=1, column=1, text="hit")],
        total=1,
        truncated=False,
    )
    assert resp.matches[0].file == "a.py"


def test_dir_entry_size_optional() -> None:
    d = DirEntry(name="a.txt", type="file", size=1234)
    assert d.size == 1234
    d2 = DirEntry(name="sub", type="dir")
    assert d2.size is None


def test_dir_request_default_include_hidden_false() -> None:
    req = DirRequest()
    assert req.include_hidden is False


def test_dir_response_basic() -> None:
    resp = DirResponse(entries=[], total=0, truncated=False)
    assert resp.total == 0


def test_tree_request_max_depth_optional() -> None:
    req = TreeRequest()
    assert req.max_depth is None


def test_tree_response_contains_rendered() -> None:
    resp = TreeResponse(rendered="a\n├── b\n", total_nodes=2, truncated=False)
    assert "├──" in resp.rendered


def test_error_envelope_matches_go_format() -> None:
    """Must match the {'error': {'code': ..., 'message': ..., 'details'?: ...}} envelope."""
    env = ErrorEnvelope(
        error=Error(
            code=ErrorCode.IO_ERROR,
            message="failure",
            details={"at": "read"},
        )
    )
    as_dict = json.loads(env.model_dump_json(exclude_none=True))
    assert as_dict == {
        "error": {
            "code": "io_error",
            "message": "failure",
            "details": {"at": "read"},
        }
    }
