# ADR-003: Resiliência — retry com backoff, DLQ e idempotência

**Status**: Aceito
**Data**: 2026-09-28
**Autores**: Time FIAP X (Fase 5 — Pós-Tech em Arquitetura de Software, FIAP)

---

## Contexto

Os consumidores (`processing-worker` e `notification-service`) precisam ser
robustos diante de falhas transitórias (banco/storage/SMTP momentaneamente
indisponíveis) sem **perder mensagens** e sem **reprocessar** trabalho já
concluído. Antes desta decisão:

- O worker fazia `Nack(requeue=false)` em falha → a mensagem era **descartada**.
- O notification-service fazia `Nack(requeue=true)` → **hot-loop** sem backoff.

## Decisão

Adotamos um padrão de **retry com backoff + Dead Letter Queue (DLQ) +
idempotência** em ambos os consumidores, via topologia RabbitMQ:

```
video.processing ──falha──▶ video.processing.retry (TTL 5s) ──expira──▶ video.events ──▶ video.processing
      │                                                                        (re-tentativa, x-retry-count+1)
      └──após 3 tentativas──▶ video.processing.dlq
```

Elementos:

1. **Fila de retry com TTL** — `*.retry` com `x-message-ttl` (backoff fixo de
   5s) e `x-dead-letter-exchange` apontando de volta ao exchange principal.
   A mensagem "dorme" 5s e volta à fila principal.
2. **Contador de tentativas** — header `x-retry-count` incrementado a cada
   retry; preservado no dead-lettering.
3. **DLQ** — após `maxRetries` (3), a mensagem vai para `*.dlq` (retenção para
   inspeção/retentativa manual).
4. **Idempotência** (worker) — antes de processar, o worker consulta o status
   atual: se `COMPLETED` ou `FAILED` (estados terminais) ou vídeo inexistente,
   ele **pula** (Ack). Isso torna segura a redelivery após crash.

## Justificativa

- **Não perder mensagem** — nada é descartado; falhas vão para DLQ.
- **Evitar hot-loop** — backoff de 5s entre tentativas.
- **Redelivery segura** — a idempotência evita reprocessar (ffmpeg é caro) e
  evitar e-mails duplicados.
- **Observabilidade** — a DLQ permite inspecionar/reprocessar manualmente.

## Consequências

- **Positivas**: entrega resiliente, sem perda nem loops; idempotência.
- **Negativas**: complexidade de topologia; backoff fixo (não exponencial);
  DLQ não reprocessa automaticamente (requer ação manual ou ferramenta).

## Relações

- **ADR-002** define o barramento (RabbitMQ/coreografia); este ADR define a
  política de entrega/resiliência sobre ele.
