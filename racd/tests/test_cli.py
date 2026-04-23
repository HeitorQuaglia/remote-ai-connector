"""Tests for the CLI. We invoke `main` with sys.argv monkey-patched."""

from __future__ import annotations

import re
from pathlib import Path

import pytest
import yaml

from racd.__main__ import main


def test_genkey_prints_32_byte_base64url(capsys: pytest.CaptureFixture[str]) -> None:
    rc = main(["genkey"])
    assert rc == 0
    out = capsys.readouterr().out.strip()
    assert re.fullmatch(r"[A-Za-z0-9_\-]+=*", out) is not None
    assert len(out) >= 40  # 32 bytes base64url ≈ 43 chars


def test_genkey_updates_config_when_named(tmp_path: Path) -> None:
    cfg = tmp_path / "config.yaml"
    cfg.write_text(
        yaml.safe_dump(
            {
                "listen": "127.0.0.1:8443",
                "api_keys": [{"name": "existing", "key": "old"}],
                "executor": {"host": "127.0.0.1", "port": 7001},
            }
        )
    )
    rc = main(["genkey", "--config", str(cfg), "--name", "new-client"])
    assert rc == 0
    data = yaml.safe_load(cfg.read_text())
    names = [e["name"] for e in data["api_keys"]]
    assert "new-client" in names


def test_run_with_missing_config_returns_error(
    tmp_path: Path, capsys: pytest.CaptureFixture[str]
) -> None:
    rc = main(["--config", str(tmp_path / "nope.yaml")])
    assert rc != 0
    err = capsys.readouterr().err
    assert "not found" in err.lower() or "no such" in err.lower()


def test_help_exits_zero(capsys: pytest.CaptureFixture[str]) -> None:
    with pytest.raises(SystemExit) as exc_info:
        main(["--help"])
    assert exc_info.value.code == 0
