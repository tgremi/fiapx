-- =============================================================
-- FIAP X — Script de criação do banco de dados (entregável)
-- Schemas por serviço: auth.* e video.*
-- Em runtime, cada serviço roda suas próprias migrations via golang-migrate;
-- este arquivo é o DDL consolidado de referência (aplicável via `psql -f`).
-- =============================================================

CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS video;

-- -------------------------------------------------------------
-- auth-service
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS auth.users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- -------------------------------------------------------------
-- video-api / processing-worker
-- -------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'video_status') THEN
        EXECUTE 'CREATE TYPE video.video_status AS ENUM (''PENDING'', ''PROCESSING'', ''COMPLETED'', ''FAILED'')';
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS video.videos (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL,
    status         video.video_status NOT NULL DEFAULT 'PENDING',
    original_key   VARCHAR(512) NOT NULL,
    zip_key        VARCHAR(512),
    error_message  TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_videos_user_id ON video.videos (user_id);
CREATE INDEX IF NOT EXISTS idx_videos_status   ON video.videos (status);
