package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/fiapx/video-api/internal/domain"
)

type fakeRepo struct {
	videos map[string]domain.Video
}

func newFakeRepo() *fakeRepo { return &fakeRepo{videos: map[string]domain.Video{}} }

func (f *fakeRepo) Create(ctx context.Context, v domain.Video) (domain.Video, error) {
	f.videos[v.ID] = v
	return v, nil
}

func (f *fakeRepo) UpdateStatus(ctx context.Context, id string, s domain.VideoStatus, zip, msg string) error {
	v := f.videos[id]
	v.Status = s
	v.ZipKey = zip
	v.ErrorMessage = msg
	f.videos[id] = v
	return nil
}

func (f *fakeRepo) FindByID(ctx context.Context, id string) (*domain.Video, error) {
	v, ok := f.videos[id]
	if !ok {
		return nil, domain.ErrVideoNotFound
	}
	return &v, nil
}

func (f *fakeRepo) ListByUser(ctx context.Context, userID string) ([]domain.Video, error) {
	var out []domain.Video
	for _, v := range f.videos {
		if v.UserID == userID {
			out = append(out, v)
		}
	}
	return out, nil
}

type fakeStorage struct {
	data map[string][]byte
}

func newFakeStorage() *fakeStorage { return &fakeStorage{data: map[string][]byte{}} }

func (s *fakeStorage) Upload(key string, data io.Reader, size int64) error {
	b, _ := io.ReadAll(data)
	s.data[key] = b
	return nil
}

func (s *fakeStorage) Open(key string) (io.ReadCloser, error) {
	b, ok := s.data[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

type fakeCache struct {
	data map[string][]byte
}

func newFakeCache() *fakeCache { return &fakeCache{data: map[string][]byte{}} }

func (c *fakeCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, ok := c.data[key]
	return v, ok, nil
}

func (c *fakeCache) Set(ctx context.Context, key string, v []byte, ttl time.Duration) error {
	c.data[key] = v
	return nil
}

func (c *fakeCache) Del(ctx context.Context, key string) error {
	delete(c.data, key)
	return nil
}

type fakePublisher struct {
	events []domain.Video
	err    error
}

func (p *fakePublisher) PublishVideoUploaded(ctx context.Context, v domain.Video) error {
	if p.err != nil {
		return p.err
	}
	p.events = append(p.events, v)
	return nil
}

func TestUploadSuccess(t *testing.T) {
	pub := &fakePublisher{}
	uc := NewVideoUseCase(newFakeRepo(), newFakeStorage(), newFakeCache(), pub)

	v, err := uc.Upload(context.Background(), "user-1", "video.mp4", bytes.NewReader([]byte("x")), 1)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if v.Status != domain.StatusPending {
		t.Fatalf("status = %s", v.Status)
	}
	if len(pub.events) != 1 {
		t.Fatalf("eventos publicados = %d", len(pub.events))
	}
	if v.UserID != "user-1" || v.OriginalKey == "" {
		t.Fatalf("dados inesperados: %+v", v)
	}
}

func TestUploadPublishFailure(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{err: errors.New("rabbit down")}
	uc := NewVideoUseCase(repo, newFakeStorage(), newFakeCache(), pub)

	if _, err := uc.Upload(context.Background(), "user-1", "v.mp4", bytes.NewReader([]byte("x")), 1); err == nil {
		t.Fatal("esperado erro de publish")
	}
	for _, v := range repo.videos {
		if v.Status != domain.StatusFailed {
			t.Fatalf("esperado FAILED, got %s", v.Status)
		}
	}
}

func TestGetByIDOwnership(t *testing.T) {
	uc := NewVideoUseCase(newFakeRepo(), newFakeStorage(), newFakeCache(), &fakePublisher{})

	v, _ := uc.Upload(context.Background(), "user-1", "v.mp4", bytes.NewReader([]byte("x")), 1)

	if _, err := uc.GetByID(context.Background(), v.ID, "user-2"); err != domain.ErrVideoNotFound {
		t.Fatalf("esperado ErrVideoNotFound, got %v", err)
	}
	if _, err := uc.GetByID(context.Background(), v.ID, "user-1"); err != nil {
		t.Fatalf("dono deveria acessar: %v", err)
	}
}

func TestDownloadPending(t *testing.T) {
	uc := NewVideoUseCase(newFakeRepo(), newFakeStorage(), newFakeCache(), &fakePublisher{})

	v, _ := uc.Upload(context.Background(), "user-1", "v.mp4", bytes.NewReader([]byte("x")), 1)

	if _, _, err := uc.Download(context.Background(), v.ID, "user-1"); err != domain.ErrVideoNotFound {
		t.Fatalf("esperado ErrVideoNotFound (PENDING), got %v", err)
	}
}
