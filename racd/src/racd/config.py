"""Configuration loading. racd reads a single YAML file whose path is passed
via the CLI. No environment overrides in V1 — keep it explicit.
"""

from __future__ import annotations

from pathlib import Path
from typing import Any

import yaml
from pydantic import BaseModel, ConfigDict, Field, field_validator


class ApiKey(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    name: str
    key: str


class ExecutorConfig(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    host: str = "127.0.0.1"
    port: int = Field(..., ge=1, le=65535)


class Config(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)

    listen: str
    api_keys: list[ApiKey]
    executor: ExecutorConfig

    @field_validator("api_keys")
    @classmethod
    def _require_non_empty(cls, v: list[ApiKey]) -> list[ApiKey]:
        if not v:
            raise ValueError("api_keys must contain at least one entry")
        return v

    def find_api_key(self, key_value: str) -> str | None:
        """Return the name of the api key that matches key_value, or None."""
        for entry in self.api_keys:
            if entry.key == key_value:
                return entry.name
        return None


def load_config(path: str | Path) -> Config:
    p = Path(path)
    with p.open("r") as f:
        raw: Any = yaml.safe_load(f)
    if not isinstance(raw, dict):
        raise ValueError(f"config root must be a mapping, got {type(raw).__name__}")
    return Config.model_validate(raw)
