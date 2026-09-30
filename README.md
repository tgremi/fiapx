# FIAP X — Processador de Vídeos (Fase 5 · Hackathon)

Sistema de processamento de vídeos da FIAP X. A partir do upload de um vídeo, o
sistema extrai frames e devolve um `.zip` para download.

Este repositório implementa a re-arquitetura do projeto base (`projeto-fiapx`,
um monolito Go) em **microsserviços**, aplicando os conceitos do curso de
Arquitetura de Software.

## Arquitetura

- **Estilo distribuído:** Microsserviços + Arquitetura Orientada a Eventos
  (coreografia via RabbitMQ) + Database-per-Service.
- **Estilo interno:** Arquitetura Hexagonal (Ports & Adapters) em cada serviço.

```
 Cliente ─▶ video-api (JWT) ─▶ RabbitMQ ─▶ processing-worker ─▶ Storage
              │   │  │                       │   │
              │   │  └▶ Storage (download)    │   └▶ PostgreSQL (status)
              ▼   ▼                          ▼
        PostgreSQL  Redis(cache)    notification-service ─▶ e-mail (MailHog)
        auth-service (register/login → JWT)
```

> **Storage (object storage):** abstraído atrás da porta `ObjectStorage`. Em dev
> usa **S3 via SeaweedFS** (Docker); em produção basta apontar para S3 real/MinIO
> sem tocar no domínio. O adaptador `FilesystemStorage` continua disponível
> (`STORAGE_DRIVER=filesystem`).

### Serviços

| Serviço | Porta | Responsabilidade |
|---|---|---|
| `auth-service` | 8081 | Cadastro/login, emissão de JWT |
| `video-api` | 8080 | Upload, listagem de status por usuário, download |
| `processing-worker` | — | Consome fila, roda ffmpeg, zipa frames, salva no storage, atualiza status |
| `notification-service` | — | Consome evento de erro, envia e-mail |

### Infraestrutura

PostgreSQL · Redis · RabbitMQ · Storage S3 (SeaweedFS) ·
MailHog (SMTP) · Prometheus · Grafana — orquestrados por **Docker Compose**
(com manifests Kubernetes em `infra/k8s/`).

## Estrutura do repositório

```
fiapx/
├─ services/
│  ├─ auth-service/
│  ├─ video-api/
│  ├─ processing-worker/
│  └─ notification-service/
├─ infra/
│  ├─ db/init.sql              # script de criação do banco (entregável)
│  ├─ monitoring/prometheus.yml
│  └─ k8s/                     # manifests Kubernetes
├─ docs/
│  ├─ adr/                     # Architecture Decision Records (ADR-001..004)
│  ├─ architecture/c4.md       # diagramas C4 (contexto/container/componente)
│  └─ video-roteiro.md         # roteiro do vídeo de apresentação (≤10 min)
├─ scripts/
│  └─ smoke-test.sh            # smoke test end-to-end (fluxo feliz + falha + S3)
├─ docker-compose.yml
└─ Makefile
```

Cada serviço segue a estrutura hexagonal:

```
service/
├─ cmd/main.go            # entrypoint (composição de dependências)
├─ internal/
│  ├─ domain/             # entidades + portas (interfaces)
│  ├─ application/        # casos de uso
│  └─ adapters/
│     ├─ in/              # driving: rest (http) / amqp (consumer)
│     └─ out/             # driven: postgres, redis, storage, rabbitmq, smtp, ffmpeg
```

## Como executar (dev)

### Pré-requisitos

- **Docker** + **Docker Compose** (v2) — caminho recomendado, sobe tudo em containers.
- Alternativa (rodar os serviços na máquina): **Go 1.27+**, `make` e `ffmpeg`.

### Subindo tudo com Docker (recomendado)

```bash
cp .env.example .env   # opcional (já existem defaults)
make up                # sobe Postgres, Redis, RabbitMQ, S3 (SeaweedFS), MailHog, Prometheus, Grafana + os 4 serviços
make smoke             # smoke test end-to-end (fluxo feliz + falha + verificação no S3)
make logs              # acompanha os logs
make down              # derruba tudo
```

URLs / portas:

| Componente | URL |
|---|---|
| `video-api` | http://localhost:8080 |
| `auth-service` | http://localhost:8081 |
| `processing-worker` (métricas) | http://localhost:9092/metrics |
| `notification-service` (métricas) | http://localhost:9093/metrics |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 (admin/admin) |
| MailHog (UI) | http://localhost:8025 |
| S3 (SeaweedFS) | http://localhost:8333 |

### Rodando os serviços na máquina (via Go)

```bash
make infra   # sobe só a infraestrutura (postgres, redis, rabbitmq, s3, mailhog, ...)
make run     # roda os 4 serviços com 'go run' (aponta para localhost)
```

Para rodar um único serviço isolado:

```bash
cd services/auth-service
go run ./cmd
```

## OS e arquitetura

As imagens base (`golang:1.27-alpine`, `alpine:3.20` + `ffmpeg`) e a infraestrutura
(Postgres, Redis, RabbitMQ, SeaweedFS, Prometheus, Grafana) são **multi-arch** —
funcionam em **amd64 (Intel)** e **arm64 (Apple Silicon/Raspberry Pi)**, nos SOs
**Linux, macOS e Windows** (Docker Desktop / WSL2).

> **Atenção:** a imagem `mailhog/mailhog` é **amd64-only** (projeto descontinuado).
> Para resolver, o `make` **detecta a arquitetura** (`uname -m`) e monta o compose
> com o override correto: `docker-compose.amd64.yml` (MailHog) ou
> `docker-compose.arm64.yml` (Mailpit, multi-arch). Use `make arch` para ver a
> arquitetura detectada.

> Para gerar **imagens multi-arch** dos serviços (publicação):
> `docker buildx build --platform linux/amd64,linux/arm64 -t fiapx/<serviço>:latest ./services/<serviço>`

## Testes

```bash
make test                # testes unitários (rápidos, sem Docker) + cobertura
make test-integration    # testes de integração (Testcontainers: Postgres real, exige Docker)
```

Os testes de integração ficam atrás da build tag `integration` e sobem um
Postgres real via Testcontainers para validar os adaptadores de persistência de
cada serviço (`auth.users`, `video.videos`), mantendo `make test`/`make cover`
rápidos e sem dependência de Docker.

## Observabilidade

Prometheus (scrape) + Grafana (dashboards) via Docker Compose:

- **Prometheus** — http://localhost:9090
- **Grafana** — http://localhost:3000 (admin/admin), dashboard "FIAP X"

Métricas expostas por serviço:

| Serviço | Endpoint | Métricas de negócio |
|---|---|---|
| auth-service | `:8081/metrics` | `auth_registrations_total`, `auth_logins_total` |
| video-api | `:8080/metrics` | `videos_uploaded_total`, `videos_downloaded_total` |
| processing-worker | `:9092/metrics` | `videos_processed_total{status}`, `video_processing_duration_seconds` |
| notification-service | `:9093/metrics` | `emails_sent_total`, `emails_failed_total` |

> Os workers escutam em `:9090`/`:9091` **dentro** da rede Docker (o Prometheus
> faz scrape nesses endpoints). No host, eles são expostos em `:9092`/`:9093`.
> Use `make metrics` para consultar os valores atuais de todas as métricas.

Todos os serviços HTTP também expõem `http_requests_total` e
`http_request_duration_seconds`. O Grafana já vem provisionado com o datasource
do Prometheus e o dashboard `fiapx.json`.

## Kubernetes & CI/CD

Manifests em `infra/k8s/` (namespace, configmap/secret, infra com SeaweedFS/S3
e os 4 serviços):

```bash
kubectl apply -f infra/k8s/
```

> O storage usa **S3 (SeaweedFS)** — o serviço `s3` sobe no próprio cluster. Em
> produção, aponte `S3_ENDPOINT` para S3 real/MinIO. Para rodar local no
> kind/minikube: `docker build -t fiapx/auth-service:latest ./services/auth-service`
> + `kind load docker-image`.

CI/CD no GitHub Actions com **esteiras separadas por serviço**
(`.github/workflows/ci-<serviço>.yml`), cada uma com **lint + test + coverage** e
gatilho por `paths` (`services/<serviço>/**`). Como os serviços são módulos Go
independentes (ADR-005), a esteira de um roda só quando aquele serviço muda.

## Status do projeto

| Fase | Descrição | Status |
|---|---|---|
| 1 | Monorepo, docker-compose, esqueletos hexagonais, ADRs | ✅ |
| 2 | auth-service (register/login + JWT) | ✅ |
| 3 | video-api (upload/listagem/download) | ✅ |
| 4 | processing-worker (ffmpeg + zip + storage) | ✅ |
| 5 | notification-service (e-mail) | ✅ |
| 6 | Resiliência (DLQ/retry/idempotência) | ✅ |
| 7 | Testes (unit + integração) | ✅ |
| 8 | Observabilidade (Prometheus/Grafana) | ✅ |
| 9 | CI/CD + manifests K8s | ✅ |
| 10 | Documentação final (C4/ADRs) + vídeo | ✅ |

## Entregáveis do desafio

- [x] Documentação da arquitetura proposta (`docs/` — ADRs + C4)
- [x] Script de criação do banco (`infra/db/init.sql`)
- [x] Roteiro do vídeo (`docs/video-roteiro.md`)
- [ ] Link do GitHub
- [ ] Vídeo gravado (≤ 10 min)

> **Nota:** o caminho do módulo Go usa `github.com/fiapx/<servico>`. Ao criar o
> repositório real, substitua `fiapx` pelo seu namespace no GitHub.
