"""OpenAPI/REST facade for ChatGPT Custom GPT Actions and any plain HTTP
client. Routes mirror the tool names at ``/v1/<tool>``. API Key auth is
applied per-route.
"""

from __future__ import annotations

from fastapi import APIRouter, Depends, FastAPI, Request, status
from fastapi.responses import JSONResponse

from racd.auth import AuthPrincipal, require_api_key
from racd.config import Config
from racd.core import ToolAdapter
from racd.executor_client import ExecutorError
from racd.schemas import (
    DirRequest,
    DirResponse,
    Error,
    ErrorCode,
    ErrorEnvelope,
    GrepRequest,
    GrepResponse,
    ReadRequest,
    ReadResponse,
    TreeRequest,
    TreeResponse,
)

_STATUS_FOR_CODE: dict[ErrorCode, int] = {
    ErrorCode.NOT_FOUND: status.HTTP_404_NOT_FOUND,
    ErrorCode.IS_DIRECTORY: status.HTTP_400_BAD_REQUEST,
    ErrorCode.BINARY_FILE: status.HTTP_400_BAD_REQUEST,
    ErrorCode.DENIED_BY_POLICY: status.HTTP_403_FORBIDDEN,
    ErrorCode.FILE_TOO_LARGE: status.HTTP_400_BAD_REQUEST,
    ErrorCode.INVALID_REGEX: status.HTTP_400_BAD_REQUEST,
    ErrorCode.INVALID_ARGUMENT: status.HTTP_400_BAD_REQUEST,
    ErrorCode.IO_ERROR: status.HTTP_500_INTERNAL_SERVER_ERROR,
    ErrorCode.EXECUTOR_UNAVAILABLE: status.HTTP_502_BAD_GATEWAY,
    ErrorCode.EXECUTOR_TIMEOUT: status.HTTP_504_GATEWAY_TIMEOUT,
}


def install_exception_handler(app: FastAPI) -> None:
    @app.exception_handler(ExecutorError)
    async def _handle(_: Request, exc: ExecutorError) -> JSONResponse:
        return JSONResponse(
            status_code=_STATUS_FOR_CODE.get(exc.code, 500),
            content=ErrorEnvelope(
                error=Error(code=exc.code, message=exc.message, details=exc.details)
            ).model_dump(exclude_none=True),
        )


def build_router(adapter: ToolAdapter, config: Config) -> APIRouter:
    router = APIRouter()
    auth = Depends(require_api_key(config))

    @router.post("/read", response_model=ReadResponse)
    async def read(req: ReadRequest, _: AuthPrincipal = auth) -> ReadResponse:
        return await adapter.read(req)

    @router.post("/grep", response_model=GrepResponse)
    async def grep(req: GrepRequest, _: AuthPrincipal = auth) -> GrepResponse:
        return await adapter.grep(req)

    @router.post("/dir", response_model=DirResponse)
    async def dir_(req: DirRequest, _: AuthPrincipal = auth) -> DirResponse:
        return await adapter.dir_(req)

    @router.post("/tree", response_model=TreeResponse)
    async def tree(req: TreeRequest, _: AuthPrincipal = auth) -> TreeResponse:
        return await adapter.tree(req)

    return router
