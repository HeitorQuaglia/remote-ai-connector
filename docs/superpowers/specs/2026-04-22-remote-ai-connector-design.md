# Remote AI Connector — Design V1

**Data:** 2026-04-22
**Status:** Design aprovado, pronto para plano de implementação.

## 1. Objetivo

Levar comportamento de agentes locais de codificação (tipo Claude Code, Codex) para dentro das interfaces web das IAs mais populares (Claude.ai e ChatGPT na V1), permitindo que esses modelos capturem contexto específico da codebase do usuário diretamente do seu dispositivo local.

O modelo enxerga a codebase em modo **read-only** e propõe soluções no chat; o usuário aplica localmente. Nada de escrita, execução, ou mutação na V1.

## 2. Posicionamento e escopo

- **Self-hosted, uso pessoal na V1.** Abrir como open source é um objetivo futuro, mas V1 não invest tempo em packaging para distribuição: build local, execução manual, tudo dimensionado para o autor do projeto.
- **Clientes suportados na V1**: Claude.ai (via Connector / MCP remoto) e ChatGPT (via Custom GPT Actions / OpenAPI).
- **Fora da V1**: Gemini e outras IAs; qualquer operação que não seja leitura; OAuth; UI web; auto-update; distribuição pública (PyPI, GitHub Releases, binários pré-compilados).

## 3. Arquitetura

### 3.1. Componentes

1. **`racd`** — servidor Python, hospedado em um VPS do usuário. Expõe dois endpoints HTTPS sobre a mesma aplicação ASGI (FastAPI):
   - `/mcp` — Streamable HTTP/SSE falando protocolo MCP, para Claude.ai Connector.
   - `/v1/*` + `/openapi.json` — REST tradicional com spec OpenAPI, para ChatGPT Custom GPT Actions.
   - Autenticação: **API Key** (header `Authorization: Bearer <key>`), uma por cliente registrado.
2. **`racx`** — binário Go, roda no laptop do usuário. Lançado a partir do terminal (`cd <projeto> && racx`). Estabelece um **reverse SSH tunnel** contra `racd` e serve, no lado do VPS via loopback, um pequeno HTTP/JSON server com 4 endpoints: `/read`, `/grep`, `/dir`, `/tree`. Autenticação: keypair **ed25519** gerenciado pelo próprio SSH.

### 3.2. Fluxo de uma chamada

```text
Claude.ai                                VPS do usuário                    Laptop do usuário
──────                                   ──────────────                    ─────────────────
POST /mcp  ───── TLS ─────►  racd (Python, porta 443)
                             │
                             │ HTTP loopback via reverse tunnel
                             ▼
                        127.0.0.1:<porta>  ◄─── reverse tunnel ────  racx (Go, porta local aleatória)
                                                                      │
                                                                      │ leitura de arquivos no CWD
                                                                      ▼
                                                                 filesystem do usuário (read-only)

                             ◄──────── resposta JSON ──────┤
◄───── resposta MCP ─────────┤
```

A "proatividade" do executor (abrir conexão para fora) é satisfeita naturalmente pelo SSH: `racx` é sempre o originador, `racd` nunca tenta alcançar o laptop.

### 3.3. Relação executor ↔ servidor

- **1:1**: um `racd` atende exatamente um `racx` simultaneamente. Múltiplos laptops ou múltiplos projetos exigem múltiplos servidores. Não há roteamento, não há seleção.

## 4. Transporte entre `racd` e `racx`

### 4.1. Camada SSH

- `racx` embute um cliente SSH nativo em Go (`golang.org/x/crypto/ssh`) — binário self-contained, sem dependência do `ssh` do sistema.
- **Primeira execução:** `racx init` gera um par ed25519 em `~/.config/remote-ai-connector/id_ed25519`, imprime a public key, e instrui o usuário a adicioná-la ao `authorized_keys` do `racd`.
- **Execução normal:** `racx` lê config (host, porta SSH, user, porta remota do tunnel), abre conexão SSH, autentica com a chave, e pede um **reverse tunnel**: "expor a porta X do seu lado apontando para a porta Y do meu lado".
- **Reconexão:** backoff exponencial (1s, 2s, 4s, …, cap em 30s). Cada reconexão renegocia o tunnel.

### 4.2. Camada aplicação (dentro do túnel)

- `racx` escuta em `127.0.0.1:0` (porta aleatória, escolhida pelo kernel). Serve HTTP/1.1 com JSON, 5 rotas: `POST /read`, `POST /grep`, `POST /dir`, `POST /tree` (as 4 tools), e `GET /ping` (healthcheck trivial, responde `{"ok": true}`).
- Status 200 para sucesso; 4xx/5xx com `{error: {code, message, details?}}` para falhas.
- `racd`, via config, sabe a porta loopback do VPS em que o tunnel está exposto (`127.0.0.1:7001` por default). Cada chamada MCP/Action que chega é traduzida para uma chamada HTTP contra esse endereço.
- **Sem autenticação adicional dentro do túnel.** O SSH já autenticou; a porta é loopback no VPS.

### 4.3. Concorrência

- Chamadas são read-only e independentes. `racx` atende em goroutines via `net/http` stdlib. Sem locks, sem fila.

### 4.4. Executor desconectado

- `racd` mantém healthcheck leve (`GET /ping` contra o tunnel). Falhas → `executor_unavailable` com mensagem acionável ao modelo.

## 5. Sandbox e segurança do filesystem

### 5.1. Raiz

- **Root = `os.Getwd()` no momento do launch do `racx`.** Imutável para o tempo de vida do processo. Não há config, não há tool dinâmica para mudar o root.

### 5.2. Validação de path

Para todo `path` recebido nas tools:

1. Resolver contra o root (concatenar, normalizar `..`).
2. Chamar `filepath.EvalSymlinks` (seguir symlinks).
3. Validar que o resultado ainda tem o root como prefixo.
4. Se não tem, erro `denied_by_policy: path escapes project root`.

### 5.3. Filtros de visibilidade

- Respeita `.gitignore` (parser compatível com git).
- Dotfiles (arquivos/diretórios começando com `.`) são **ocultos por default**. Parâmetro `include_hidden: true` nas tools permite override.
- **Denylist hardcoded não-contornável** (aplica mesmo com `include_hidden: true`):
  - `.env`, `.env.*`
  - `*.pem`, `*.key`, `*.crt`
  - `*_rsa`, `*_rsa.pub`, `*_ed25519`, `*_ed25519.pub`, `*_ecdsa`, `*_ecdsa.pub`, `*_dsa`, `*_dsa.pub`
  - `.git/objects/`, `.git/refs/`, `.git/HEAD` (o resto de `.git/` também não vaza por default, mas o bloqueio desses três é estrutural)
  - `.ssh/`, `.aws/credentials`, `.gnupg/`

Qualquer tentativa de `read`/`grep`/`dir`/`tree` que bata na denylist retorna `denied_by_policy`.

## 6. Superfície das tools

Todas as tools compartilham a mesma definição para MCP e OpenAPI; o servidor Python gera as duas fachadas a partir de uma única fonte (Pydantic).

### 6.1. `read(path, offset?, limit?)`

- Lê arquivo texto, `path` relativo ao root.
- Default: linha 1, até 2000 linhas ou 256KB (o que vier primeiro).
- `offset`: linha inicial, 1-indexed. `limit`: número máximo de linhas.
- **Response**: `{ content, total_lines, truncated, returned_range: {start, end} }`.
- **Erros**: `not_found`, `is_directory`, `binary_file` (null byte nos primeiros 8KB), `denied_by_policy`, `file_too_large` (se `total_bytes > 10MB`), `io_error`.

### 6.2. `grep(pattern, path?, include?, context_lines?, include_hidden?)`

- Regex **RE2** (Go `regexp`, sem risco de ReDoS).
- `path`: subárvore para varrer (default: root).
- `include`: glob de arquivos (ex: `**/*.py`).
- `context_lines`: 0..5, default 0.
- Respeita `.gitignore` + dotfiles + denylist.
- **Response**: `{ matches: [{file, line, column, text, before, after}], total, truncated }`, máx 200 matches.
- **Erros**: `invalid_regex`, `denied_by_policy`, `io_error`.

### 6.3. `dir(path?, include_hidden?)`

- Listagem não-recursiva de um diretório (default: root).
- **Response**: `{ entries: [{name, type: "file"|"dir"|"symlink", size?, symlink_target?}], total, truncated }`, máx 500 entradas, ordenadas (dirs primeiro, alfa).

### 6.4. `tree(path?, max_depth?, include_hidden?)`

- Árvore recursiva a partir de `path` (default: root).
- `max_depth`: default 3, máx 10.
- Respeita `.gitignore` + dotfiles + denylist.
- **Response** (JSON): `{ rendered: string, total_nodes: int, truncated: bool }`, onde `rendered` é a árvore formatada estilo `tree(1)` (com `├──`, `└──`, `│`). Aproximadamente 5000 nós máximo.

### 6.5. Formato de erro consistente

```json
{ "error": { "code": "denied_by_policy", "message": "Path escapes project root", "details": {} } }
```

Códigos aparecem também na description de cada tool no manifest MCP / OpenAPI para que o modelo saiba o que esperar e como reagir.

## 7. Estrutura do repositório

```text
remote-ai-connector/
├── racd/                        # servidor Python
│   ├── pyproject.toml
│   ├── src/racd/
│   │   ├── __main__.py          # entrypoint: `python -m racd`
│   │   ├── app.py               # ASGI app (FastAPI) monta /mcp e /v1
│   │   ├── mcp_facade.py        # adapter MCP → core
│   │   ├── openapi_facade.py    # adapter REST/OpenAPI → core
│   │   ├── core.py              # 4 funções: read/grep/dir/tree que chamam o executor
│   │   ├── executor_client.py   # HTTP client para 127.0.0.1:<tunnel>
│   │   ├── auth.py              # API key middleware
│   │   ├── config.py            # carrega config (YAML/env)
│   │   └── schemas.py           # Pydantic models (fonte única de verdade)
│   └── tests/
│       ├── unit/                # mocka executor, testa fachadas
│       └── integration/         # sobe racd + racx fake via socket local
├── racx/                        # executor Go
│   ├── go.mod
│   ├── cmd/racx/main.go         # entrypoint: flags, init, run
│   ├── internal/
│   │   ├── sshtunnel/           # cliente SSH + reverse tunnel + reconexão
│   │   ├── fs/                  # resolução de path, sandbox, denylist, gitignore
│   │   ├── tools/               # read.go, grep.go, dir.go, tree.go
│   │   ├── server/              # HTTP server local (net/http)
│   │   └── audit/               # logging formatado em stderr
│   └── tests/                   # testes Go, tabela + temp dir fixtures
├── docs/
│   └── superpowers/specs/       # specs deste brainstorm
├── examples/
│   └── racd-config.yaml.example # config de exemplo para referência pessoal
├── README.md                    # notas de uso para o autor
└── concept.md                   # preservado como contexto histórico
```

## 8. Configuração e execução

### 8.1. `racd`

- Execução via `python -m racd --config /etc/racd/config.yaml` (ou caminho arbitrário do config).
- Instalação local via `pip install -e racd/` a partir do clone do repo. Sem publicação em PyPI na V1.
- Subcomandos CLI do `racd`:
  - `racd genkey [--name <alias>]` — gera uma nova API key aleatória (32 bytes base64url), imprime-a e grava no `config.yaml` atual sob `api_keys`.
  - `racd --config <path>` (sem subcomando) — roda o servidor.

### 8.2. `racx`

- Build local via `go build ./racx/cmd/racx` (ou `go install` dentro do próprio checkout). Sem binários pré-compilados, sem GoReleaser na V1.
- Execução: `cd <projeto> && racx` (usando o binário compilado).

### 8.3. `racd` (`config.yaml`)

```yaml
listen: "0.0.0.0:8443"
tls:
  cert: /etc/racd/cert.pem
  key:  /etc/racd/key.pem
api_keys:
  - name: "claude-ai"
    key:  "..."       # gerado via `racd genkey`
  - name: "chatgpt"
    key:  "..."
ssh:
  authorized_keys: /etc/racd/authorized_keys
  listen: "0.0.0.0:2222"
  host_key: /etc/racd/ssh_host_ed25519_key
executor:
  tunnel_port: 7001   # porta loopback que recebe o reverse tunnel
```

### 8.4. `racx` (`~/.config/remote-ai-connector/config.yaml` + flags)

```yaml
server:
  host: racd.exemplo.com
  ssh_port: 2222
  ssh_user: racx
identity: ~/.config/remote-ai-connector/id_ed25519
remote_tunnel_port: 7001
local_listen: "127.0.0.1:0"
```

Flags principais: `--config`, `--log-file`, `--quiet`, `init` (subcomando para gerar chave).

## 9. Observabilidade

### 9.1. `racx` — audit trail em stderr (default)

```text
[14:32:01] ✓ tunnel up  racd.exemplo.com:2222
[14:32:04] read  src/app.py                    (1.2KB, 42 lines)
[14:32:07] grep  "class User"  **/*.py         (12 matches)
[14:32:11] tree  .  depth=3                    (187 nodes)
[14:32:40] ✗ read  /etc/passwd                 (denied_by_policy: outside root)
```

- `--log-file <path>`: adiciona log JSON estruturado em arquivo.
- `--quiet`: silencia stderr.

### 9.2. `racd`

- Logs JSON estruturados em stdout, com `request_id`, `tool`, `api_key_name`, `latency_ms`, `status`.
- `/healthz` (sem auth) — disponível; `/metrics` Prometheus não faz parte da V1.

## 10. Tratamento de erros

- **Erros locais do `racx`** (não encontrado, denylist, binário): `{error: {code, message}}` com HTTP 4xx; `racd` repassa.
- **Erros de transporte** (tunnel caiu, timeout de 30s): `executor_unavailable` ou `executor_timeout` com mensagem acionável.
- **Erros de auth** (API key ruim no `racd`, SSH key ruim no `racx`): 401/403 claros.
- **Nunca** expor stack traces ao modelo: sanitização obrigatória na fachada.

## 11. Testes

- **`racd`**: pytest. Unit (fachadas com executor mockado) + integration (ASGI + executor fake in-process via socket local).
- **`racx`**: `go test`. Tabela de casos para sandbox (path traversal, symlink escape, denylist) como prioridade máxima; fixtures de filesystem via `t.TempDir()`.
- **E2E** em `tests/e2e/`: sobe `racd` em Docker + `racx` em subprocesso, exercita `/mcp` e `/v1/*`. Não roda em CI por default; execução sob demanda.

## 12. CI (GitHub Actions)

- Lint: `ruff` + `mypy` (Python); `go vet` + `staticcheck` (Go).
- Testes unitários e de integração para os dois componentes.
- `go build` para validar que o `racx` compila em Linux/macOS.
- **Sem release automatizado, sem publicação.** CI apenas garante que a branch principal permanece em estado buildável e com testes verdes.

## 13. Riscos e mitigações

1. **Prompt injection convencendo o modelo a ler secrets** → root fixo + denylist hardcoded não-desabilitável.
2. **Symlink escape do root** → `filepath.EvalSymlinks` + validação de prefixo após a resolução.
3. **Vazamento de API key** → rotação manual via `racd genkey` + edição do config (V1).
4. **`racd` comprometido** → pior caso é leitura do projeto (tools read-only e sandboxed). VPS é zona de confiança do usuário.
5. **Secrets incidentais no código do projeto** → responsabilidade do usuário; documentar no README.
6. **DoS via regex/glob amplo** → limites de tamanho/matches já aplicados; timeout de 30s por chamada no `racx`.

## 14. Explicitamente fora da V1

- Escrita, execução, mutação de qualquer tipo (`Write`, `Bash`, `Edit`, criação de diretório).
- Múltiplos executores por servidor.
- Múltiplos projetos por executor.
- Multi-usuário / multi-tenancy no `racd`.
- OAuth (só API key estática).
- Streaming de respostas (paginação síncrona para arquivos grandes).
- UI web (CLI/config only).
- Integração com Gemini ou outras IAs.
- Auto-update do `racx`.
- Distribuição pública: PyPI, GitHub Releases, GoReleaser, binários pré-compilados, Dockerfile, unit systemd de exemplo — nada disso é V1. Tudo é build local e execução manual pelo autor.
- Suporte Windows para o `racx` (V1 valida Linux e macOS).

## 15. Critérios de sucesso da V1

1. O autor do projeto consegue subir o `racd` no próprio VPS e conectar o `racx` a partir de um laptop em menos de 15 minutos, seguindo anotações do próprio repo.
2. Claude.ai consegue, via Connector, responder perguntas sobre uma codebase real usando as 4 tools.
3. ChatGPT Custom GPT consegue o mesmo via Actions.
4. Tentativas de ler `/etc/passwd`, `~/.ssh/id_rsa` ou `../../etc/shadow` retornam `denied_by_policy`.
5. `.env` na raiz do projeto nunca é lido, mesmo se o modelo pedir explicitamente.
