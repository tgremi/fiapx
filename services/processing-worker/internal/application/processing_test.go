package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/fiapx/processing-worker/internal/domain"
)

type fakeProcessor struct {
	err    error
	called bool
}

func (f *fakeProcessor) Process(ctx context.Context, src io.Reader) (io.ReadCloser, error) {
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	return io.NopCloser(bytes.NewReader([]byte("zipdata"))), nil
}

type fakeStorage struct {
	data      map[string][]byte
	uploadErr error
}

func newFakeStorage() *fakeStorage { return &fakeStorage{data: map[string][]byte{}} }

func (s *fakeStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	b, ok := s.data[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (s *fakeStorage) Upload(ctx context.Context, key string, data io.Reader) error {
	if s.uploadErr != nil {
		return s.uploadErr
	}
	b, _ := io.ReadAll(data)
	s.data[key] = b
	return nil
}

type fakeRepo struct {
	current   domain.VideoStatus
	status    domain.VideoStatus
	zipKey    string
	errMsg    string
	updates   int
	findErr   error
	updateErr error
}

func newFakeRepo(initial domain.VideoStatus) *fakeRepo {
	return &fakeRepo{current: initial}
}

func (f *fakeRepo) GetStatus(ctx context.Context, id string) (domain.VideoStatus, error) {
	if f.findErr != nil {
		return "", f.findErr
	}
	return f.current, nil
}

func (f *fakeRepo) UpdateStatus(ctx context.Context, id string, s domain.VideoStatus, zipKey, errMsg string) error {
	f.updates++
	if f.updateErr != nil {
		return f.updateErr
	}
	f.status = s
	f.zipKey = zipKey
	f.errMsg = errMsg
	f.current = s
	return nil
}

type fakePublisher struct {
	failed []string
}

func (f *fakePublisher) PublishVideoFailed(ctx context.Context, job domain.ProcessingJob, errMsg string) error {
	f.failed = append(f.failed, job.VideoID)
	return nil
}

func TestProcessSuccess(t *testing.T) {
	storage := newFakeStorage()
	storage.data["u1/v1.mp4"] = []byte("video")
	repo := newFakeRepo(domain.StatusPending)
	processor := &fakeProcessor{}
	uc := NewProcessingUseCase(processor, storage, repo, &fakePublisher{})

	job := domain.ProcessingJob{VideoID: "v1", UserID: "u1", OriginalKey: "u1/v1.mp4"}
	if err := uc.Process(context.Background(), job); err != nil {
		t.Fatalf("process: %v", err)
	}

	if repo.status != domain.StatusCompleted {
		t.Fatalf("status = %s", repo.status)
	}
	if repo.zipKey != "u1/v1.zip" {
		t.Fatalf("zipKey = %s", repo.zipKey)
	}
	if !processor.called {
		t.Fatal("processador deveria ter sido chamado")
	}
}

func TestProcessFfmpegFailure(t *testing.T) {
	storage := newFakeStorage()
	storage.data["u1/v1.mp4"] = []byte("video")
	repo := newFakeRepo(domain.StatusPending)
	pub := &fakePublisher{}
	uc := NewProcessingUseCase(&fakeProcessor{err: errors.New("ffmpeg bug")}, storage, repo, pub)

	job := domain.ProcessingJob{VideoID: "v1", UserID: "u1", OriginalKey: "u1/v1.mp4"}
	if err := uc.Process(context.Background(), job); err != nil {
		t.Fatalf("falha permanente não deve retornar erro para retry: %v", err)
	}

	if repo.status != domain.StatusFailed {
		t.Fatalf("status = %s", repo.status)
	}
	if len(pub.failed) != 1 {
		t.Fatalf("eventos de falha = %d", len(pub.failed))
	}
}

func TestProcessDownloadFailureIsRetryable(t *testing.T) {
	repo := newFakeRepo(domain.StatusPending)
	pub := &fakePublisher{}
	uc := NewProcessingUseCase(&fakeProcessor{}, newFakeStorage(), repo, pub)

	job := domain.ProcessingJob{VideoID: "v1", UserID: "u1", OriginalKey: "u1/v1.mp4"}
	if err := uc.Process(context.Background(), job); err == nil {
		t.Fatal("falha transitória de download deve retornar erro (retry)")
	}

	if repo.status == domain.StatusFailed {
		t.Fatal("falha transitória não deve marcar FAILED")
	}
	if len(pub.failed) != 0 {
		t.Fatal("falha transitória não deve notificar")
	}
}

func TestProcessIdempotentCompleted(t *testing.T) {
	repo := newFakeRepo(domain.StatusCompleted)
	processor := &fakeProcessor{}
	uc := NewProcessingUseCase(processor, newFakeStorage(), repo, &fakePublisher{})

	job := domain.ProcessingJob{VideoID: "v1", UserID: "u1", OriginalKey: "u1/v1.mp4"}
	if err := uc.Process(context.Background(), job); err != nil {
		t.Fatalf("process: %v", err)
	}

	if processor.called {
		t.Fatal("não deveria reprocessar vídeo já COMPLETED")
	}
	if repo.updates != 0 {
		t.Fatalf("não deveria atualizar status, updates = %d", repo.updates)
	}
}

func TestProcessIdempotentFailed(t *testing.T) {
	repo := newFakeRepo(domain.StatusFailed)
	processor := &fakeProcessor{}
	uc := NewProcessingUseCase(processor, newFakeStorage(), repo, &fakePublisher{})

	job := domain.ProcessingJob{VideoID: "v1", UserID: "u1", OriginalKey: "u1/v1.mp4"}
	if err := uc.Process(context.Background(), job); err != nil {
		t.Fatalf("process: %v", err)
	}
	if processor.called {
		t.Fatal("não deveria reprocessar vídeo já FAILED")
	}
}

func TestProcessVideoNotFound(t *testing.T) {
	repo := newFakeRepo(domain.StatusPending)
	repo.findErr = domain.ErrVideoNotFound
	processor := &fakeProcessor{}
	uc := NewProcessingUseCase(processor, newFakeStorage(), repo, &fakePublisher{})

	job := domain.ProcessingJob{VideoID: "v1", UserID: "u1", OriginalKey: "u1/v1.mp4"}
	if err := uc.Process(context.Background(), job); err != nil {
		t.Fatalf("esperado skip silencioso, got %v", err)
	}
	if processor.called {
		t.Fatal("não deveria processar vídeo inexistente")
	}
}

func TestProcessUpdateStatusFailure(t *testing.T) {
	storage := newFakeStorage()
	storage.data["u1/v1.mp4"] = []byte("video")
	repo := newFakeRepo(domain.StatusPending)
	repo.updateErr = errors.New("db down")
	uc := NewProcessingUseCase(&fakeProcessor{}, storage, repo, &fakePublisher{})

	job := domain.ProcessingJob{VideoID: "v1", UserID: "u1", OriginalKey: "u1/v1.mp4"}
	if err := uc.Process(context.Background(), job); err == nil {
		t.Fatal("esperado erro transitório no update de status")
	}
	if repo.status == domain.StatusFailed {
		t.Fatal("falha transitória não deve marcar FAILED")
	}
}
