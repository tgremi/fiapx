package domain

import (
	"context"
	"errors"
	"io"
	"time"
)

var ErrVideoNotFound = errors.New("video not found")

type VideoStatus string

const (
	StatusPending    VideoStatus = "PENDING"
	StatusProcessing VideoStatus = "PROCESSING"
	StatusCompleted  VideoStatus = "COMPLETED"
	StatusFailed     VideoStatus = "FAILED"
)

type Video struct {
	ID           string
	UserID       string
	Status       VideoStatus
	OriginalKey  string
	ZipKey       string
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Claims struct {
	UserID string
	Email  string
}

type TokenValidator interface {
	Validate(token string) (Claims, error)
}

type VideoRepository interface {
	Create(ctx context.Context, v Video) (Video, error)
	UpdateStatus(ctx context.Context, id string, status VideoStatus, zipKey, errMsg string) error
	FindByID(ctx context.Context, id string) (*Video, error)
	ListByUser(ctx context.Context, userID string) ([]Video, error)
}

type ObjectStorage interface {
	Upload(key string, data io.Reader, size int64) error
	Open(key string) (io.ReadCloser, error)
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

type EventPublisher interface {
	PublishVideoUploaded(ctx context.Context, v Video) error
}
