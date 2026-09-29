# ADR-002: Comunicação assíncrona via EDA + Coreografia (RabbitMQ)

**Status**: Aceito
**Data**: 2026-09-28
**Autores**: Time FIAP X (Fase 5 — Pós-Tech em Arquitetura de Software, FIAP)

---

## Contexto

Requisitos funcionais do desafio:

- O sistema deve processar **mais de um vídeo ao mesmo tempo** e, **em picos,
  não perder requisição**.
- Em caso de erro, o usuário deve ser **notificado** (e-mail).

O monolito original processa o vídeo de forma **síncrona** dentro do handler
HTTP: em um pico, as requisições se acumulam/estouram e não há notificação de
erro.

Precisamos de um mecanismo que desacople o recebimento do upload do
processamento, garanta entrega sob carga e conecte o processamento à
notificação.

## Decisão

Adotamos **Arquitetura Orientada a Eventos (EDA)** com **coreografia** usando
**RabbitMQ** como broker. Não há orquestrador central; cada serviço reage aos
eventos de seu interesse.

Fluxo e eventos:

```
upload ─▶ video-api ─▶ [video.uploaded] ─▶ processing-worker ─▶ [video.failed]
                                                 │                       │
                                                 └──▶ atualiza status    └─▶ notification-service ─▶ e-mail
```

| Evento | Produtor | Consumidor | Efeito |
|---|---|---|---|
| `video.uploaded` | `video-api` | `processing-worker` | dispara o processamento |
| `video.failed` | `processing-worker` | `notification-service` | envia e-mail de erro |

Padrões de entrega/resiliência aplicados:

- **Queue-Based Load Leveling** — a fila absorve picos; a API responde
  **202 Accepted** e enfileira.
- **Competing Consumers** — N instâncias do worker na mesma fila processam em
  paralelo.
- **Idempotent Consumer** — worker tolerante a duplicidade.
- **DLQ + retry/backoff** — mensagens com falha vão para uma Dead Letter Queue.

## Justificativa

- **Escalabilidade** — escala horizontal do worker independente da API.
- **Resiliência** — fila desacopla e preserva requisições sob pico (requisito).
- **Simplicidade** — coreografia é mais simples que orquestração (Saga) para um
  fluxo linear de 2 etapas; não há transação distribuída que exija compensação.
- **Conhecimento prévio** — o time já operou RabbitMQ na Fase 4 (MechFlow).

> Nota: na Fase 4 o time usou **Saga Orquestrado**. Aqui o fluxo é um pipeline
> linear e idempotente, então a coreografia é a escolha adequada.

## Alternativas consideradas

- **Kafka** — robusto para streaming/alto volume, porém maior complexidade
  operacional (tópicos, partições, consumer groups) desproporcional ao escopo.
- **Orquestração (Saga)** — desnecessária sem transações distribuídas.
- **Processamento síncrono** — descartado (perde requisições em pico).

## Consequências

- **Positivas**: desacoplamento, escalabilidade, entrega resiliente, satisfaz o
  requisito de picos e notificação.
- **Negativas**: eventual consistência de status (o cliente consulta o estado
  via polling/listagem); necessidade de idempotência e tratamento de DLQ.

## Relações

- **ADR-001** define a estrutura interna (Hexagonal) de cada serviço; este ADR
  define como os serviços se comunicam.
