CREATE SCHEMA IF NOT EXISTS video;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'video_status') THEN
        EXECUTE 'CREATE TYPE video.video_status AS ENUM (''PENDING'', ''PROCESSING'', ''COMPLETED'', ''FAILED'')';
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS video.videos (
    id            UUID PRIMARY KEY,
    user_id       UUID NOT NULL,
    status        video.video_status NOT NULL DEFAULT 'PENDING',
    original_key  VARCHAR(512) NOT NULL,
    zip_key       VARCHAR(512),
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_videos_user_id ON video.videos (user_id);
CREATE INDEX IF NOT EXISTS idx_videos_status ON video.videos (status);
