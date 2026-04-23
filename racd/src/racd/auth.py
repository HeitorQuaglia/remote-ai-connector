"""API Key authentication for both facades (MCP + OpenAPI).

The dependency reads ``Authorization: Bearer <key>`` and validates against
``config.api_keys``. On success it yields an ``AuthPrincipal`` naming the
key so downstream logging can attribute the request.
"""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from typing import Annotated

from fastapi import FastAPI, Header, Request
from fastapi.responses import JSONResponse

from racd.config import Config
from racd.schemas import Error, ErrorCode, ErrorEnvelope


@dataclass(frozen=True)
class AuthPrincipal:
    name: str


class ApiKeyError(Exception):
    """Custom exception for API key authentication failures."""

    def __init__(self, message: str) -> None:
        super().__init__(message)
        self.message = message


def require_api_key(config: Config) -> Callable[..., AuthPrincipal]:
    """Return a FastAPI dependency bound to the given config."""

    def _dep(
        authorization: Annotated[str | None, Header()] = None,
    ) -> AuthPrincipal:
        if not authorization or not authorization.startswith("Bearer "):
            _reject("missing or malformed Authorization header")
        # After the check above, authorization is guaranteed to be a str
        assert authorization is not None
        token = authorization.removeprefix("Bearer ").strip()
        name = config.find_api_key(token)
        if name is None:
            _reject("invalid API key")
        assert name is not None
        return AuthPrincipal(name=name)

    return _dep


def _reject(message: str) -> None:
    raise ApiKeyError(message)


def install_handler(app: FastAPI) -> None:
    """Install the exception handler for ApiKeyError in the given FastAPI app."""

    @app.exception_handler(ApiKeyError)
    async def handle_api_key_error(request: Request, exc: ApiKeyError) -> JSONResponse:
        return JSONResponse(
            status_code=401,
            content=ErrorEnvelope(
                error=Error(code=ErrorCode.INVALID_ARGUMENT, message=exc.message)
            ).model_dump(exclude_none=True),
        )
