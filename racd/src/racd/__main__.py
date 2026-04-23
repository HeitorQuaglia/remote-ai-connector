"""CLI entry point for racd.

Usage::

    python -m racd --config /etc/racd/config.yaml
    python -m racd genkey [--config PATH] [--name ALIAS]

The default (no subcommand) starts the server via uvicorn.
"""

from __future__ import annotations

import argparse
import secrets
import sys
from collections.abc import Sequence
from pathlib import Path

import yaml

from racd.config import load_config


def _gen_token() -> str:
    # 32 bytes base64url, no padding. Equivalent to Go's base64.RawURLEncoding.
    import base64

    return base64.urlsafe_b64encode(secrets.token_bytes(32)).rstrip(b"=").decode("ascii")


def _cmd_genkey(args: argparse.Namespace) -> int:
    token = _gen_token()
    print(token)
    if args.config:
        path = Path(args.config)
        data = yaml.safe_load(path.read_text())
        if not isinstance(data, dict):
            print(f"racd: config at {path} is not a mapping", file=sys.stderr)
            return 1
        keys = data.setdefault("api_keys", [])
        if not isinstance(keys, list):
            print("racd: api_keys must be a list", file=sys.stderr)
            return 1
        keys.append({"name": args.name or f"key-{len(keys) + 1}", "key": token})
        path.write_text(yaml.safe_dump(data, sort_keys=False))
    return 0


def _cmd_run(args: argparse.Namespace) -> int:
    import uvicorn

    if not args.config:
        print("racd: --config is required", file=sys.stderr)
        return 2

    try:
        config = load_config(args.config)
    except FileNotFoundError as e:
        print(f"racd: config not found: {e}", file=sys.stderr)
        return 1

    from racd.app import build_app

    app = build_app(config)

    host, _, port = config.listen.partition(":")
    if not port:
        print(f"racd: invalid listen address {config.listen!r}", file=sys.stderr)
        return 1

    uvicorn.run(app, host=host, port=int(port), log_level="info")
    return 0


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="racd", description="Remote AI Connector server")
    parser.add_argument("--config", help="Path to racd config YAML")
    sub = parser.add_subparsers(dest="command")

    gk = sub.add_parser("genkey", help="Generate a new API key")
    gk.add_argument("--config", help="Append key to this config file")
    gk.add_argument("--name", help="Alias for the generated key (default: key-N)")
    gk.set_defaults(func=_cmd_genkey)

    parser.set_defaults(func=_cmd_run)
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    parser = _build_parser()
    args = parser.parse_args(argv)
    return int(args.func(args))


if __name__ == "__main__":
    raise SystemExit(main())
