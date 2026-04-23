"""Tests for YAML-based config loading."""

from __future__ import annotations

from pathlib import Path

import pytest

from racd.config import ApiKey, ExecutorConfig, load_config


def _write(tmp_path: Path, content: str) -> Path:
    p = tmp_path / "config.yaml"
    p.write_text(content)
    return p


def test_load_minimal_config(tmp_path: Path) -> None:
    path = _write(
        tmp_path,
        """
listen: "127.0.0.1:8443"
api_keys:
  - name: claude-ai
    key: abc123
executor:
  host: 127.0.0.1
  port: 7001
""",
    )
    cfg = load_config(path)
    assert cfg.listen == "127.0.0.1:8443"
    assert cfg.api_keys == [ApiKey(name="claude-ai", key="abc123")]
    assert cfg.executor == ExecutorConfig(host="127.0.0.1", port=7001)


def test_multiple_api_keys(tmp_path: Path) -> None:
    path = _write(
        tmp_path,
        """
listen: "0.0.0.0:8443"
api_keys:
  - name: claude-ai
    key: key-claude
  - name: chatgpt
    key: key-chatgpt
executor:
  host: 127.0.0.1
  port: 7001
""",
    )
    cfg = load_config(path)
    assert len(cfg.api_keys) == 2
    assert cfg.api_keys[1].name == "chatgpt"


def test_api_keys_must_be_non_empty(tmp_path: Path) -> None:
    path = _write(
        tmp_path,
        """
listen: "127.0.0.1:8443"
api_keys: []
executor:
  host: 127.0.0.1
  port: 7001
""",
    )
    with pytest.raises(ValueError, match="at least one"):
        load_config(path)


def test_executor_port_must_be_valid(tmp_path: Path) -> None:
    path = _write(
        tmp_path,
        """
listen: "127.0.0.1:8443"
api_keys:
  - name: k
    key: v
executor:
  host: 127.0.0.1
  port: 0
""",
    )
    with pytest.raises(ValueError):
        load_config(path)


def test_api_key_lookup_by_value(tmp_path: Path) -> None:
    path = _write(
        tmp_path,
        """
listen: "127.0.0.1:8443"
api_keys:
  - name: one
    key: aaa
  - name: two
    key: bbb
executor:
  host: 127.0.0.1
  port: 7001
""",
    )
    cfg = load_config(path)
    assert cfg.find_api_key("aaa") == "one"
    assert cfg.find_api_key("bbb") == "two"
    assert cfg.find_api_key("wrong") is None


def test_missing_file_raises(tmp_path: Path) -> None:
    with pytest.raises(FileNotFoundError):
        load_config(tmp_path / "does-not-exist.yaml")


def test_config_is_immutable(tmp_path: Path) -> None:
    path = _write(
        tmp_path,
        """
listen: "127.0.0.1:8443"
api_keys:
  - name: k
    key: v
executor:
  host: 127.0.0.1
  port: 7001
""",
    )
    cfg = load_config(path)
    with pytest.raises(ValueError):
        cfg.listen = "something-else"  # type: ignore[misc]
