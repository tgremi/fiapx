# Roteiro do vídeo (≤ 10 min) — FIAP X · Processador de Vídeos

Objetivo: apresentar a re-arquitetura do monolito `projeto-fiapx` em
microsserviços, com ênfase em arquitetura, resiliência e qualidade.

## 0:00 — 0:30 · Abertura

- "FIAP X — Processador de Vídeos". Enunciado do desafio: upload de vídeo →
  extração de frames (ffmpeg) → download de `.zip`.
- O ponto de partida: um monolito Go com tudo num único `main.go` (496 linhas),
  processamento síncrono e sem notificação de erro.

## 0:30 — 2:30 · Arquitetura proposta (o "porquê")

- **Microsserviços** (4 serviços) + **Database-per-Service**.
- **Hexagonal (Ports & Adapters)** interno em cada serviço → testabilidade e
  desacoplamento (ADR-001).
- **EDA + coreografia** via RabbitMQ (ADR-002) → desacopla upload do
  processamento e absorve picos.
- **Object Storage S3** (SeaweedFS em dev) → artefatos compartilhados sem
  filesystem local (ADR-004).
- Mostrar o diagrama C4 (níveis 1–3).

## 2:30 — 4:30 · Fluxo feliz (demo)

1. `register` + `login` → JWT.
2. `POST /videos` (multipart) → **202 Accepted**.
3. `video-api` grava no S3 e publica `video.uploaded`.
4. `processing-worker` consome, roda ffmpeg (`fps=1`), zipa frames, grava no S3
   e atualiza o status (`PENDING → PROCESSING → COMPLETED`).
5. `GET /videos/:id/download` → baixa o `.zip`.

## 4:30 — 6:00 · Resiliência (ADR-003)

- Fila `.retry` com TTL + `x-retry-count` e **DLQ** após 3 tentativas.
- **Idempotência** no worker (não reprocessa COMPLETED/FAILED).
- Distinção erro **permanente** (ffmpeg) vs **transitório** (storage/DB).

## 6:00 — 7:30 · Fluxo de falha (demo)

- Upload de arquivo inválido → ffmpeg falha → status `FAILED` → evento
  `video.failed` → `notification-service` envia e-mail (MailHog).

## 7:30 — 9:00 · Qualidade e operação

- Testes unitários + integração (Testcontainers) e cobertura (`make test`,
  `make cover`, `make test-integration`).
- Observabilidade: Prometheus + Grafana (métricas por serviço).
- CI/CD (GitHub Actions) + manifests Kubernetes.

## 9:00 — 10:00 · Fechamento

- Smoke test end-to-end (`make smoke`) rodando na tela, mostrando o S3.
- Recapitular decisões (ADR-001..004) e próximos passos.
