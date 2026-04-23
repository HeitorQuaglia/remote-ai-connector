# remote-ai-connector

Ponte entre IAs web (Claude.ai, ChatGPT) e a codebase local do autor. Projeto pessoal em construção.

- Design: [`docs/superpowers/specs/2026-04-22-remote-ai-connector-design.md`](docs/superpowers/specs/2026-04-22-remote-ai-connector-design.md)
- Planos de implementação: [`docs/superpowers/plans/`](docs/superpowers/plans/)

## Componentes

- [`racx/`](racx/) — executor local em Go.
- `racd/` — servidor Python (Plano 2, ainda não implementado).

## Status

- Plano 1 (racx core): concluído.
- Plano 2 (racd core): concluído.
- Plano 3 (SSH + E2E integration): a escrever.
