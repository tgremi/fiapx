package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestUploadAndDownload(t *testing.T) {
	ctx := context.Background()
	s := NewFilesystemStorage(t.TempDir())

	data := []byte("zip dos frames")
	if err := s.Upload(ctx, "u1/v1.zip", bytes.NewReader(data)); err != nil {
		t.Fatalf("upload: %v", err)
	}

	r, err := s.Download(ctx, "u1/v1.zip")
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer func() { _ = r.Close() }()

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("dados divergem: %q != %q", got, data)
	}
}

func TestDownloadNotFound(t *testing.T) {
	s := NewFilesystemStorage(t.TempDir())
	if _, err := s.Download(context.Background(), "nao/existe.zip"); err == nil {
		t.Fatal("esperado erro ao baixar arquivo inexistente")
	}
}
