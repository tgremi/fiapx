package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/fiapx/video-api/internal/domain"
	"github.com/google/uuid"
)

type VideoUseCase struct {
	videos    domain.VideoRepository
	storage   domain.ObjectStorage
	cache     domain.Cache
	publisher domain.EventPublisher
}

func NewVideoUseCase(
	videos domain.VideoRepository,
	storage domain.ObjectStorage,
	cache domain.Cache,
	publisher domain.EventPublisher,
) *VideoUseCase {
	return &VideoUseCase{
		videos:    videos,
		storage:   storage,
		cache:     cache,
		publisher: publisher,
	}
}

func (uc *VideoUseCase) Upload(ctx context.Context, userID, filename string, data io.Reader, size int64) (domain.Video, error) {
	id := uuid.NewString()
	ext := strings.ToLower(filepath.Ext(filename))
	key := fmt.Sprintf("%s/%s%s", userID, id, ext)

	if err := uc.storage.Upload(key, data, size); err != nil {
		return domain.Video{}, err
	}

	v := domain.Video{
		ID:          id,
		UserID:      userID,
		Status:      domain.StatusPending,
		OriginalKey: key,
	}

	created, err := uc.videos.Create(ctx, v)
	if err != nil {
		return domain.Video{}, err
	}

	if err := uc.publisher.PublishVideoUploaded(ctx, created); err != nil {
		_ = uc.videos.UpdateStatus(ctx, created.ID, domain.StatusFailed, "", err.Error())
		return domain.Video{}, err
	}

	_ = uc.cache.Del(ctx, cacheKey(userID))
	return created, nil
}

func (uc *VideoUseCase) ListByUser(ctx context.Context, userID string) ([]domain.Video, error) {
	key := cacheKey(userID)
	if data, ok, err := uc.cache.Get(ctx, key); err == nil && ok {
		var vids []domain.Video
		if json.Unmarshal(data, &vids) == nil {
			return vids, nil
		}
	}

	vids, err := uc.videos.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(vids); err == nil {
		_ = uc.cache.Set(ctx, key, data, 30*time.Second)
	}

	return vids, nil
}

func (uc *VideoUseCase) GetByID(ctx context.Context, id, userID string) (*domain.Video, error) {
	v, err := uc.videos.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if v.UserID != userID {
		return nil, domain.ErrVideoNotFound
	}
	return v, nil
}

func (uc *VideoUseCase) Download(ctx context.Context, id, userID string) (io.ReadCloser, string, error) {
	v, err := uc.GetByID(ctx, id, userID)
	if err != nil {
		return nil, "", err
	}
	if v.Status != domain.StatusCompleted || v.ZipKey == "" {
		return nil, "", domain.ErrVideoNotFound
	}

	r, err := uc.storage.Open(v.ZipKey)
	if err != nil {
		return nil, "", err
	}

	return r, filepath.Base(v.ZipKey), nil
}

func cacheKey(userID string) string {
	return "videos:" + userID
}
