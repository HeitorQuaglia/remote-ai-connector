# racd

Python server for `remote-ai-connector`. Exposes two facades over the same core:

- `/mcp` — MCP Streamable HTTP, for Claude.ai Connectors.
- `/v1/*` + `/openapi.json` — REST, for ChatGPT Custom GPT Actions.

Both facades proxy calls to a locally-reachable `racx` executor over HTTP. API
key auth is enforced per request.

## Install

```bash
cd racd
python -m venv .venv && . .venv/bin/activate
pip install -e ".[dev]"
```

## Config

Generate a key and create a config file:

```bash
python -m racd genkey --config examples/racd-config.yaml.example --name claude-ai
```

Minimum config (see `examples/racd-config.yaml.example`):

```yaml
listen: "127.0.0.1:8443"
api_keys:
  - name: claude-ai
    key: <generated>
executor:
  host: 127.0.0.1
  port: 7001
```

## Run

Start `racx` first (from Plano 1) in the target project's directory:

```bash
cd racx && ./racx --listen 127.0.0.1:7001
```

Then start `racd`:

```bash
cd racd
python -m racd --config /path/to/racd-config.yaml
```

## Test

```bash
cd racd && pytest
```

The integration test in `tests/test_e2e.py` uses an in-process stub `racx`; no external processes required.

## Endpoints

| Path | Auth | Purpose |
|------|------|---------|
| `GET /healthz` | no | Liveness probe |
| `GET /openapi.json` | no | OpenAPI spec for the REST facade |
| `POST /v1/read` | Bearer | Read a file |
| `POST /v1/grep` | Bearer | Search files |
| `POST /v1/dir` | Bearer | List a directory |
| `POST /v1/tree` | Bearer | Render a directory tree |
| Streamable HTTP `/mcp/*` | Bearer | MCP protocol |

## Scope

This is Plano 2 of 3. SSH reverse-tunnel integration between `racd` and `racx` — so that `racd` can run on a VPS while `racx` runs on the author's laptop — lands in Plano 3.
