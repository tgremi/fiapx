package domain

import (
	"context"
	"errors"
	"io"
)

var ErrVideoNotFound = errors.New("video not found")

type VideoStatus string

const (
	StatusPending    VideoStatus = "PENDING"
	StatusProcessing VideoStatus = "PROCESSING"
	StatusCompleted  VideoStatus = "COMPLETED"
	StatusFailed     VideoStatus = "FAILED"
)

type ProcessingJob struct {
	VideoID     string
	UserID      string
	OriginalKey string
}

type VideoProcessor interface {
	Process(ctx context.Context, src io.Reader) (io.ReadCloser, error)
}

type ObjectStorage interface {
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Upload(ctx context.Context, key string, data io.Reader) error
}

type VideoRepository interface {
	UpdateStatus(ctx context.Context, id string, status VideoStatus, zipKey, errMsg string) error
	GetStatus(ctx context.Context, id string) (VideoStatus, error)
}

type EventPublisher interface {
	PublishVideoFailed(ctx context.Context, job ProcessingJob, errMsg string) error
}
