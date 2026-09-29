# ADR-001: Arquitetura Hexagonal (Ports & Adapters) como padrão interno

**Status**: Aceito
**Data**: 2026-09-28
**Autores**: Time FIAP X (Fase 5 — Pós-Tech em Arquitetura de Software, FIAP)

---

## Contexto

A Fase 5 (Hackathon) exige re-arquitetar o monolito Go `projeto-fiapx` em
microsserviços com qualidade de software (testes) e boas práticas. O projeto
base concentra toda a lógica em um único `main.go` (496 linhas), acoplando
diretamente HTTP (Gin), sistema de arquivos e a chamada ao ffmpeg (`os/exec`).

Precisamos de um padrão interno que:

- Isole a lógica de negócio das dependências de infraestrutura (HTTP, banco,
  fila, storage, e-mail, ffmpeg).
- Torne cada serviço **testável** sem levantar infraestrutura real.
- Permita trocar adaptadores (ex.: ffmpeg local → outro mecanismo) sem tocar
  no domínio.

## Decisão

Adotamos **Arquitetura Hexagonal (Ports & Adapters)** em **todos** os serviços
(`auth-service`, `video-api`, `processing-worker`, `notification-service`).

Regras:

1. **Domínio puro** — `internal/domain` contém entidades, value objects e
   **portas** (interfaces). Nada de Gin, amqp, pgx ou `os/exec` aqui.
2. **Casos de uso** — `internal/application` orquestra as portas.
3. **Adaptadores** — `internal/adapters/in` (driving: HTTP, consumidor AMQP) e
   `internal/adapters/out` (driven: PostgreSQL, Redis, MinIO, RabbitMQ, SMTP,
   ffmpeg).
4. **Dependência aponta para dentro** — adaptadores dependem das portas do
   domínio, nunca o contrário.
5. **Composição no entrypoint** — `cmd/main.go` injeta as implementações reais.

```
service/
├─ cmd/main.go            # entrypoint (composição)
├─ internal/
│  ├─ domain/             # entidades + portas (interfaces)
│  ├─ application/        # casos de uso
│  └─ adapters/
│     ├─ in/              # rest (http) / amqp (consumer)
│     └─ out/             # postgres, redis, minio, rabbitmq, smtp, ffmpeg
```

## Justificativa

- **Testabilidade** — o ffmpeg vira a porta `VideoProcessor`, mockável em testes
  unitários; o repositório vira a porta `VideoRepository`, testável com fake.
- **Fronteiras claras** — cada adaptador encapsula um detalhe técnico
  (S3/MinIO, AMQP, SMTP), reduzindo acoplamento.
- **Consistência com o curso** — padrão já conhecido e valorizado nos critérios
  de "Desenho de arquitetura" e "Qualidade de software".
- **Evolução** — trocar banco, storage ou protocolo não afeta o domínio.

## Consequências

- **Positivas**: testes unitários rápidos e estáveis; domínio legível; baixo
  acoplamento.
- **Negativas**: mais indireção (interfaces + wiring) que um script monolítico;
  exige disciplina para não vazar detalhes de infraestrutura para o domínio.

## Relações

- Complementa o **ADR-002** (EDA/coreografia), que define a comunicação entre
  serviços; este ADR define a estrutura interna de cada um.
