"""httpx-based client that talks to the racx executor.

The executor exposes ``/ping`` (GET, no auth) and POST endpoints per tool.
On success each endpoint returns the tool's JSON response body. On failure
the status is 4xx/5xx and the body is ``{"error": {"code", "message", ...}}``.

This client unwraps both shapes and exposes a single `ExecutorError` to
callers, so the facades can map straight to `ErrorEnvelope`.
"""

from __future__ import annotations

import json
from typing import Any, NoReturn

import httpx

from racd.schemas import (
    DirRequest,
    DirResponse,
    ErrorCode,
    GrepRequest,
    GrepResponse,
    ReadRequest,
    ReadResponse,
    TreeRequest,
    TreeResponse,
)

DEFAULT_TIMEOUT = httpx.Timeout(connect=3.0, read=30.0, write=30.0, pool=3.0)


class ExecutorError(Exception):
    def __init__(
        self,
        code: ErrorCode,
        message: str,
        details: dict[str, Any] | None = None,
    ) -> None:
        super().__init__(f"{code.value}: {message}")
        self.code = code
        self.message = message
        self.details = details


class ExecutorClient:
    def __init__(
        self,
        base_url: str,
        transport: httpx.BaseTransport | None = None,
        timeout: httpx.Timeout = DEFAULT_TIMEOUT,
    ) -> None:
        self._client = httpx.AsyncClient(
            base_url=base_url,
            timeout=timeout,
            transport=transport,  # type: ignore[arg-type]
        )

    async def aclose(self) -> None:
        await self._client.aclose()

    async def ping(self) -> bool:
        try:
            r = await self._client.get("/ping")
        except httpx.TimeoutException as e:
            raise ExecutorError(ErrorCode.EXECUTOR_TIMEOUT, str(e)) from e
        except httpx.TransportError as e:
            raise ExecutorError(ErrorCode.EXECUTOR_UNAVAILABLE, str(e)) from e
        if r.status_code != 200:
            return False
        body = _safe_json(r)
        return bool(body.get("ok", False))

    async def read(self, req: ReadRequest) -> ReadResponse:
        data = await self._post_tool("/read", req.model_dump(exclude_none=True))
        return ReadResponse.model_validate(data)

    async def grep(self, req: GrepRequest) -> GrepResponse:
        data = await self._post_tool("/grep", req.model_dump(exclude_none=True))
        return GrepResponse.model_validate(data)

    async def dir_(self, req: DirRequest) -> DirResponse:
        data = await self._post_tool("/dir", req.model_dump(exclude_none=True))
        return DirResponse.model_validate(data)

    async def tree(self, req: TreeRequest) -> TreeResponse:
        data = await self._post_tool("/tree", req.model_dump(exclude_none=True))
        return TreeResponse.model_validate(data)

    async def _post_tool(self, path: str, body: dict[str, Any]) -> dict[str, Any]:
        try:
            r = await self._client.post(path, json=body)
        except httpx.TimeoutException as e:
            raise ExecutorError(ErrorCode.EXECUTOR_TIMEOUT, str(e)) from e
        except httpx.TransportError as e:
            raise ExecutorError(ErrorCode.EXECUTOR_UNAVAILABLE, str(e)) from e
        if r.status_code == 200:
            result = _safe_json(r)
            return result
        return _raise_for_error(r)


def _safe_json(r: httpx.Response) -> dict[str, Any]:
    try:
        data = r.json()
        if not isinstance(data, dict):
            raise ExecutorError(ErrorCode.IO_ERROR, "executor response is not a JSON object")
        return data
    except json.JSONDecodeError as e:
        raise ExecutorError(ErrorCode.IO_ERROR, f"malformed executor response: {e}") from e


def _raise_for_error(r: httpx.Response) -> NoReturn:
    try:
        body = r.json()
    except json.JSONDecodeError:
        raise ExecutorError(
            ErrorCode.IO_ERROR,
            f"executor returned HTTP {r.status_code} with non-JSON body",
        ) from None
    err = body.get("error") if isinstance(body, dict) else None
    if not isinstance(err, dict):
        raise ExecutorError(
            ErrorCode.IO_ERROR,
            f"executor returned HTTP {r.status_code} without error envelope",
        )
    try:
        code = ErrorCode(err.get("code", ""))
    except ValueError:
        code = ErrorCode.IO_ERROR
    raise ExecutorError(
        code,
        str(err.get("message", "")),
        details=err.get("details") if isinstance(err.get("details"), dict) else None,
    )
