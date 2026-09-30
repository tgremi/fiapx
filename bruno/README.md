# Coleção Bruno — FIAP X

Documentação das rotas da aplicação FIAP X (Fase 5 · Hackathon).

## Como abrir

1. Instale o [Bruno](https://www.usebruno.com/) (app desktop ou CLI).
2. Abra a pasta `bruno/` como coleção (Open Collection → selecione `bruno/`).
3. Selecione o environment **Local** (canto superior direito).
4. Rode as requisições na ordem abaixo.

## Fluxo de teste

| # | Pasta | Request | Observação |
|---|---|---|---|
| 1 | auth-service | `Login` (ou `Register`) | o `Login` grava o token automaticamente em `{{token}}` |
| 2 | video-api | `Upload Vídeo` | envia `test.mp4`; grava `{{videoId}}` |
| 3 | video-api | `Status do Vídeo` | `PENDING → PROCESSING → COMPLETED/FAILED` |
| 4 | video-api | `Download do ZIP` | só funciona quando `COMPLETED` |
| 5 | video-api | `Upload Inválido` | envia `invalido.mp4`; termina em `FAILED` + e-mail de erro |

## Métricas (Prometheus)

Requests `Metrics` por serviço (sem autenticação):

| Pasta | Request | URL |
|---|---|---|
| auth-service | `Metrics` | `GET /metrics` (:8081) |
| video-api | `Metrics` | `GET /metrics` (:8080) |
| processing-worker | `Metrics` | `GET /metrics` (:9090) |
| notification-service | `Metrics` | `GET /metrics` (:9091) |

## Variáveis (environment `Local`)

| Variável | Valor |
|---|---|
| `authUrl` | `http://localhost:8081` (auth-service) |
| `videoUrl` | `http://localhost:8080` (video-api) |
| `processingMetricsUrl` | `http://localhost:9090` (processing-worker) |
| `notificationMetricsUrl` | `http://localhost:9091` (notification-service) |
| `email` / `password` | credenciais de teste |

`token` e `videoId` são preenchidos automaticamente pelos scripts `post-response`
(Login → `token`, Upload → `videoId`).

## Vídeos de teste

A coleção usa os vídeos commitados na raiz do repositório:

- `test.mp4` → upload válido (fluxo feliz)
- `invalido.mp4` → arquivo inválido (fluxo de falha → `FAILED`)

Os caminhos nas requests são relativos ao arquivo `.bru`
(`@file(../../test.mp4)` e `@file(../../invalido.mp4)`).

Para gerar um `test.mp4` alternativo:

```bash
ffmpeg -loglevel error -f lavfi -i testsrc=duration=3:size=320x240:rate=10 -pix_fmt yuv420p test.mp4 -y
```

Para usar outro arquivo, edite o campo `video` da request `Upload Vídeo`
(`@file(/caminho/do/arquivo)`).
