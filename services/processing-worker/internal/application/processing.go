package application

import (
	"context"
	"errors"
	"time"

	"github.com/fiapx/processing-worker/internal/domain"
	"github.com/fiapx/processing-worker/internal/metrics"
)

type ProcessingUseCase struct {
	processor domain.VideoProcessor
	storage   domain.ObjectStorage
	videos    domain.VideoRepository
	publisher domain.EventPublisher
}

func NewProcessingUseCase(
	processor domain.VideoProcessor,
	storage domain.ObjectStorage,
	videos domain.VideoRepository,
	publisher domain.EventPublisher,
) *ProcessingUseCase {
	return &ProcessingUseCase{
		processor: processor,
		storage:   storage,
		videos:    videos,
		publisher: publisher,
	}
}

func (uc *ProcessingUseCase) Process(ctx context.Context, job domain.ProcessingJob) error {
	start := time.Now()
	defer func() { metrics.ObserveProcessing(time.Since(start)) }()

	status, err := uc.videos.GetStatus(ctx, job.VideoID)
	if err != nil {
		if errors.Is(err, domain.ErrVideoNotFound) {
			return nil
		}
		metrics.ProcessingError()
		return err
	}

	if status == domain.StatusCompleted || status == domain.StatusFailed {
		return nil
	}

	_ = uc.videos.UpdateStatus(ctx, job.VideoID, domain.StatusProcessing, "", "")

	src, err := uc.storage.Download(ctx, job.OriginalKey)
	if err != nil {
		metrics.ProcessingError()
		return err
	}
	defer func() { _ = src.Close() }()

	zipReader, err := uc.processor.Process(ctx, src)
	if err != nil {
		uc.markFailedAndNotify(ctx, job, err)
		return nil
	}
	defer func() { _ = zipReader.Close() }()

	zipKey := job.UserID + "/" + job.VideoID + ".zip"
	if err := uc.storage.Upload(ctx, zipKey, zipReader); err != nil {
		metrics.ProcessingError()
		return err
	}

	if err := uc.videos.UpdateStatus(ctx, job.VideoID, domain.StatusCompleted, zipKey, ""); err != nil {
		metrics.ProcessingError()
		return err
	}

	metrics.ProcessedCompleted()
	return nil
}

func (uc *ProcessingUseCase) markFailedAndNotify(ctx context.Context, job domain.ProcessingJob, err error) {
	metrics.ProcessedFailed()
	_ = uc.videos.UpdateStatus(ctx, job.VideoID, domain.StatusFailed, "", err.Error())
	_ = uc.publisher.PublishVideoFailed(ctx, job, err.Error())
}
