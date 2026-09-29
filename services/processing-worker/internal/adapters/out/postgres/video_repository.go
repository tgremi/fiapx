package postgres

import (
	"context"
	"errors"

	"github.com/fiapx/processing-worker/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VideoRepository struct {
	pool *pgxpool.Pool
}

func NewVideoRepository(pool *pgxpool.Pool) *VideoRepository {
	return &VideoRepository{pool: pool}
}

func (r *VideoRepository) UpdateStatus(ctx context.Context, id string, status domain.VideoStatus, zipKey, errMsg string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE video.videos
		SET status = $2::video.video_status,
		    zip_key = NULLIF($3, ''),
		    error_message = NULLIF($4, ''),
		    updated_at = now()
		WHERE id = $1::uuid
	`, id, string(status), zipKey, errMsg)
	return err
}

func (r *VideoRepository) GetStatus(ctx context.Context, id string) (domain.VideoStatus, error) {
	var status string
	err := r.pool.QueryRow(ctx,
		`SELECT status::text FROM video.videos WHERE id = $1::uuid`,
		id,
	).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrVideoNotFound
		}
		return "", err
	}
	return domain.VideoStatus(status), nil
}
