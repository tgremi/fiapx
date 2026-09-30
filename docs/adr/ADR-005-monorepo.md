# ADR-005: Monorepo com módulos Go independentes

**Status**: Aceito
**Data**: 2026-09-29
**Autores**: Time FIAP X (Fase 5 — Pós-Tech em Arquitetura de Software, FIAP)

---

## Contexto

O projeto é composto por 4 serviços (`auth-service`, `video-api`,
`processing-worker`, `notification-service`) que precisam evoluir juntos,
compartilhando infraestrutura (Docker Compose, Kubernetes, Prometheus/Grafana),
documentação (ADRs, C4) e tooling (`Makefile`, smoke test).

Era preciso decidir entre:

- **Polyrepo** — um repositório Git por serviço.
- **Monorepo** — um único repositório Git contendo todos os serviços e artefatos
  compartilhados.

## Decisão

Adotamos **monorepo**: um único repositório com a estrutura:

```
fiapx/
├─ services/
│  ├─ auth-service/          # módulo Go próprio
│  ├─ video-api/             # módulo Go próprio
│  ├─ processing-worker/     # módulo Go próprio
│  └─ notification-service/  # módulo Go próprio
├─ infra/                    # docker-compose, k8s, monitoring, s3
├─ docs/                     # ADRs, C4
├─ scripts/                  # smoke test
├─ docker-compose.yml
└─ Makefile
```

Cada serviço é um **módulo Go independente**: possui o próprio `go.mod`
(`module github.com/fiapx/<serviço>`), dependências e build/testes isolados, e
**não importa código de outros serviços** (comunicação exclusivamente por
eventos RabbitMQ e HTTP/JWT — ver ADR-002).

## Justificativa

- **Facilidade de estruturação e correção do projeto** — em um único repositório
  é possível navegar, corrigir e revisar o sistema inteiro de uma vez; correções
  que envolvem serviço + infraestrutura + documentação viram um único
  commit/PR (mudança atômica).
- **Fonte única de infra e docs** — `infra/`, `docs/` e `Makefile` não precisam
  ser sincronizados entre múltiplos repositórios.
- **Módulos independentes** — como não há dependência de código entre serviços,
  cada um compila e testa isoladamente, permitindo **esteiras de CI separadas
  por serviço** (com gatilho por `paths` em `services/<serviço>/**`).

## Consequências

- **Positivas**: visão única do todo; mudanças atômicas entre serviço e infra;
  setup e onboarding simplificados; CI independente por serviço.
- **Negativas**: histórico Git compartilhado; necessidade de disciplina nos
  gatilhos de CI (filtro por `paths`) para não rodar todas as esteiras a cada
  push.

## Relações

- **ADR-002** — a independência entre serviços (comunicação por eventos) é o que
  torna viável o monorepo com módulos isolados e esteiras separadas.
- **ADR-001** — cada serviço segue a estrutura hexagonal interna.
