package application

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/fiapx/video-api/internal/domain"
)

func TestListByUserCacheMiss(t *testing.T) {
	repo := newFakeRepo()
	cache := newFakeCache()
	repo.videos["vid-1"] = domain.Video{ID: "vid-1", UserID: "u-1", Status: domain.StatusPending}

	uc := NewVideoUseCase(repo, newFakeStorage(), cache, &fakePublisher{})

	got, err := uc.ListByUser(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].ID != "vid-1" {
		t.Fatalf("list inesperada: %+v", got)
	}
	if _, ok := cache.data["videos:u-1"]; !ok {
		t.Fatal("cache deveria ter sido populado")
	}
}

func TestListByUserCacheHit(t *testing.T) {
	repo := newFakeRepo()
	cache := newFakeCache()

	v := domain.Video{ID: "vid-1", UserID: "u-1", Status: domain.StatusCompleted}
	data, _ := json.Marshal([]domain.Video{v})
	cache.data["videos:u-1"] = data

	uc := NewVideoUseCase(repo, newFakeStorage(), cache, &fakePublisher{})

	got, err := uc.ListByUser(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Status != domain.StatusCompleted {
		t.Fatalf("list inesperada: %+v", got)
	}
}

func TestDownloadCompleted(t *testing.T) {
	repo := newFakeRepo()
	storage := newFakeStorage()
	repo.videos["vid-1"] = domain.Video{
		ID: "vid-1", UserID: "u-1", Status: domain.StatusCompleted, ZipKey: "u-1/vid-1.zip",
	}
	storage.data["u-1/vid-1.zip"] = []byte("zip-content")

	uc := NewVideoUseCase(repo, storage, newFakeCache(), &fakePublisher{})

	r, name, err := uc.Download(context.Background(), "vid-1", "u-1")
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer func() { _ = r.Close() }()

	if name != "vid-1.zip" {
		t.Fatalf("filename = %q", name)
	}
	b, _ := io.ReadAll(r)
	if string(b) != "zip-content" {
		t.Fatalf("conteúdo = %q", b)
	}
}

func TestDownloadStorageError(t *testing.T) {
	repo := newFakeRepo()
	repo.videos["vid-1"] = domain.Video{
		ID: "vid-1", UserID: "u-1", Status: domain.StatusCompleted, ZipKey: "u-1/vid-1.zip",
	}

	uc := NewVideoUseCase(repo, newFakeStorage(), newFakeCache(), &fakePublisher{})

	if _, _, err := uc.Download(context.Background(), "vid-1", "u-1"); err == nil {
		t.Fatal("esperado erro ao abrir zip ausente no storage")
	}
}
