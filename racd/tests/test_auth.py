"""Tests for the API Key FastAPI dependency."""

from __future__ import annotations

from fastapi import Depends, FastAPI
from fastapi.testclient import TestClient

from racd.auth import AuthPrincipal, install_handler, require_api_key
from racd.config import ApiKey, Config, ExecutorConfig


def _cfg(*keys: tuple[str, str]) -> Config:
    return Config(
        listen="127.0.0.1:0",
        api_keys=[ApiKey(name=n, key=k) for n, k in keys],
        executor=ExecutorConfig(host="127.0.0.1", port=7001),
    )


def _app(cfg: Config) -> FastAPI:
    app = FastAPI()
    install_handler(app)

    @app.get("/protected")
    def protected(principal: AuthPrincipal = Depends(require_api_key(cfg))) -> dict[str, str]:  # noqa: B008
        return {"name": principal.name}

    return app


def test_missing_header_returns_401() -> None:
    app = _app(_cfg(("k", "secret")))
    client = TestClient(app)
    r = client.get("/protected")
    assert r.status_code == 401
    assert r.json()["error"]["code"] == "invalid_argument"


def test_wrong_key_returns_401() -> None:
    app = _app(_cfg(("k", "secret")))
    client = TestClient(app)
    r = client.get("/protected", headers={"Authorization": "Bearer wrong"})
    assert r.status_code == 401


def test_valid_key_returns_principal_name() -> None:
    app = _app(_cfg(("claude-ai", "correct")))
    client = TestClient(app)
    r = client.get("/protected", headers={"Authorization": "Bearer correct"})
    assert r.status_code == 200
    assert r.json() == {"name": "claude-ai"}


def test_non_bearer_scheme_rejected() -> None:
    app = _app(_cfg(("k", "secret")))
    client = TestClient(app)
    r = client.get("/protected", headers={"Authorization": "Basic c2VjcmV0"})
    assert r.status_code == 401


def test_multiple_keys_any_valid() -> None:
    app = _app(_cfg(("claude-ai", "one"), ("chatgpt", "two")))
    client = TestClient(app)
    r = client.get("/protected", headers={"Authorization": "Bearer two"})
    assert r.status_code == 200
    assert r.json() == {"name": "chatgpt"}
