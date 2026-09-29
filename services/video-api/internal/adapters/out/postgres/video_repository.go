package postgres

import (
	"context"
	"errors"

	"github.com/fiapx/video-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VideoRepository struct {
	pool *pgxpool.Pool
}

func NewVideoRepository(pool *pgxpool.Pool) *VideoRepository {
	return &VideoRepository{pool: pool}
}

func (r *VideoRepository) Create(ctx context.Context, v domain.Video) (domain.Video, error) {
	var status string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO video.videos (id, user_id, status, original_key)
		VALUES ($1::uuid, $2::uuid, $3::video.video_status, $4)
		RETURNING id::text, user_id::text, status::text, original_key, created_at, updated_at
	`, v.ID, v.UserID, string(v.Status), v.OriginalKey).Scan(
		&v.ID, &v.UserID, &status, &v.OriginalKey, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return domain.Video{}, err
	}
	v.Status = domain.VideoStatus(status)
	return v, nil
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

func (r *VideoRepository) FindByID(ctx context.Context, id string) (*domain.Video, error) {
	var v domain.Video
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, user_id::text, status::text, original_key,
		       COALESCE(zip_key, ''), COALESCE(error_message, ''), created_at, updated_at
		FROM video.videos
		WHERE id = $1::uuid
	`, id).Scan(&v.ID, &v.UserID, &status, &v.OriginalKey, &v.ZipKey, &v.ErrorMessage, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVideoNotFound
		}
		return nil, err
	}
	v.Status = domain.VideoStatus(status)
	return &v, nil
}

func (r *VideoRepository) ListByUser(ctx context.Context, userID string) ([]domain.Video, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text, status::text, original_key,
		       COALESCE(zip_key, ''), COALESCE(error_message, ''), created_at, updated_at
		FROM video.videos
		WHERE user_id = $1::uuid
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vids := make([]domain.Video, 0)
	for rows.Next() {
		var v domain.Video
		var status string
		if err := rows.Scan(&v.ID, &v.UserID, &status, &v.OriginalKey, &v.ZipKey, &v.ErrorMessage, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		v.Status = domain.VideoStatus(status)
		vids = append(vids, v)
	}
	return vids, rows.Err()
}
