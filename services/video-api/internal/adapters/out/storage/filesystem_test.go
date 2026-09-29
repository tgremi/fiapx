package storage

import (
	"bytes"
	"io"
	"testing"
)

func TestUploadAndOpen(t *testing.T) {
	s := NewFilesystemStorage(t.TempDir())

	data := []byte("conteudo do video")
	if err := s.Upload("u1/v1/video.mp4", bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("upload: %v", err)
	}

	r, err := s.Open("u1/v1/video.mp4")
	if err != nil {
		t.Fatalf("open: %v", err)
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

func TestOpenNotFound(t *testing.T) {
	s := NewFilesystemStorage(t.TempDir())
	if _, err := s.Open("nao/existe.mp4"); err == nil {
		t.Fatal("esperado erro ao abrir arquivo inexistente")
	}
}
