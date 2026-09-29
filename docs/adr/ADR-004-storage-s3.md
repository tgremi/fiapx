# ADR-004: Object Storage S3-compatível (SeaweedFS) via minio-go

**Status**: Aceito
**Data**: 2026-09-29
**Autores**: Time FIAP X (Fase 5 — Pós-Tech em Arquitetura de Software, FIAP)

---

## Contexto

O sistema precisa persistir o **vídeo original** (upload) e o **zip de frames**
(processamento) em um *object storage*, permitindo que `video-api` e
`processing-worker` compartilhem os artefatos de forma desacoplada e escalável.

Inicialmente o projeto usou um **volume local compartilhado** (filesystem) e,
mais tarde, tentamos **MinIO** como storage S3-compatível. Porém:

- As imagens `minio/minio`, `quay.io/minio/minio` e `bitnami/minio` **saíram dos
  registries** (Docker Hub / Quay), inviabilizando o MinIO local.
- `localstack/localstack:latest` passou a **exigir licença paga** (exit 55 —
  `License activation failed`).
- O **AWS SDK for Go v2** apresentou incompatibilidade com o gateway S3 do
  SeaweedFS (`not found: ComputePayloadHash`).

## Decisão

Adotamos **SeaweedFS** (S3-compatível, open source, imagem disponível) como
object storage de desenvolvimento, usando o client **minio-go v7** (leve,
estável e feito para storage S3-compatível).

- O storage continua atrás da porta `domain.ObjectStorage` (Upload/Open em
  `video-api`; Download/Upload em `processing-worker`).
- Novo adaptador `S3Storage` implementa a porta nos dois serviços.
- Autenticação S3 configurada via `infra/s3/s3.json` (identidade `test`/`test`).
- `STORAGE_DRIVER` seleciona `s3` (padrão) ou `filesystem` (fallback), mantendo
  o adaptador `FilesystemStorage` funcional e testado.

## Justificativa

- **Padrão de mercado** — S3 é o protocolo *de facto* para object storage; em
  produção basta apontar `S3_ENDPOINT`/credenciais para S3 real sem tocar no
  domínio (troca de adaptador).
- **Desacoplamento** — produtor e consumidor não compartilham filesystem local;
  escala horizontal em K8s sem volume `ReadWriteMany`.
- **minio-go** — API simples (`PutObject`/`GetObject`/`BucketExists`), suporta
  chunked upload e funciona com SeaweedFS/MinIO/S3.

## Alternativas consideradas

- **MinIO** — descartado (imagens indisponíveis).
- **LocalStack** — descartado (licença obrigatória na imagem `latest`).
- **AWS SDK v2** — descartado por incompatibilidade com o gateway S3 do
  SeaweedFS (`ComputePayloadHash`).
- **Volume compartilhado (filesystem)** — mantido como fallback
  (`STORAGE_DRIVER=filesystem`), mas não escala em múltiplos nós.

## Consequências

- **Positivas**: storage desacoplado e escalável; troca transparente para S3
  real; sem dependência de volume RWX no K8s.
- **Negativas**: mais um componente de infraestrutura (SeaweedFS) e dependência
  do client `minio-go`; o adaptador `S3Storage` só é coberto em runtime/integração
  (não em testes unitários).

## Relações

- **ADR-001** — o `S3Storage` é mais um adaptador *driven* da porta
  `ObjectStorage`, dentro da estrutura hexagonal.
- **ADR-002** — o storage é compartilhado via eventos (upload → processamento),
  sem acoplamento direto entre `video-api` e `processing-worker`.
