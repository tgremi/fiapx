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
| 2 | video-api | `Upload Vídeo` | envia `/tmp/test.mp4`; grava `{{videoId}}` |
| 3 | video-api | `Status do Vídeo` | `PENDING → PROCESSING → COMPLETED/FAILED` |
| 4 | video-api | `Download do ZIP` | só funciona quando `COMPLETED` |

## Variáveis (environment `Local`)

| Variável | Valor |
|---|---|
| `authUrl` | `http://localhost:8081` (auth-service) |
| `videoUrl` | `http://localhost:8080` (video-api) |
| `email` / `password` | credenciais de teste |

`token` e `videoId` são preenchidos automaticamente pelos scripts `post-response`
(Login → `token`, Upload → `videoId`).

## Vídeo de teste

O upload usa `/tmp/test.mp4`. Gere um vídeo de teste com:

```bash
ffmpeg -loglevel error -f lavfi -i testsrc=duration=3:size=320x240:rate=10 -pix_fmt yuv420p /tmp/test.mp4 -y
```

Para usar outro arquivo, edite o campo `video` da request `Upload Vídeo`
(`@file(/caminho/do/arquivo)`).
