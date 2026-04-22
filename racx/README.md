# racx

Executor local do `remote-ai-connector`. Expõe 4 tools read-only (Read, Grep, Dir, Tree) sobre um servidor HTTP/JSON local.

Esta versão (Plano 1) ainda não inclui o SSH reverse tunnel — isso vem no Plano 3. Por enquanto, o servidor escuta em `127.0.0.1:<porta>` localmente e pode ser exercitado com `curl`.

## Build

```bash
cd racx
go build -o racx ./cmd/racx
```

## Uso

```bash
cd /caminho/do/projeto
racx --listen 127.0.0.1:7777
```

Em outro terminal:

```bash
curl -s http://127.0.0.1:7777/ping
curl -s -X POST http://127.0.0.1:7777/read -d '{"path":"README.md"}'
curl -s -X POST http://127.0.0.1:7777/grep -d '{"pattern":"TODO"}'
curl -s -X POST http://127.0.0.1:7777/dir -d '{}'
curl -s -X POST http://127.0.0.1:7777/tree -d '{"max_depth":2}'
```

## Flags

- `--listen host:port` — endereço de escuta (default `127.0.0.1:0`, porta aleatória).
- `--quiet` — suprime o audit trail em stderr.
- `--print-port` — imprime o endereço de escuta em stdout (útil para automação).

## Sandbox e segurança

- Root é o CWD no momento da execução; imutável depois.
- Path traversal (`..`, paths absolutos fora, symlinks que escapam) é bloqueado.
- `.gitignore` da raiz é respeitado; dotfiles ocultos por default (override via `include_hidden: true` por chamada).
- Denylist hardcoded (`.env*`, chaves SSH, `.git/objects`, etc.) não pode ser desabilitada.

## Testes

```bash
cd racx && go test ./...
```

Smoke test de ponta-a-ponta (compila o binário e exercita `/ping`):

```bash
cd racx && RACX_SMOKE=1 go test ./cmd/racx/... -run Smoke -v
```
