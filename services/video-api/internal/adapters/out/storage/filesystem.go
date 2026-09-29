package storage

import (
	"io"
	"os"
	"path/filepath"
)

type FilesystemStorage struct {
	root string
}

func NewFilesystemStorage(root string) *FilesystemStorage {
	_ = os.MkdirAll(root, 0o755)
	return &FilesystemStorage{root: root}
}

func (s *FilesystemStorage) Upload(key string, data io.Reader, size int64) error {
	path := filepath.Join(s.root, filepath.Clean(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(f, data)
	return err
}

func (s *FilesystemStorage) Open(key string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.root, filepath.Clean(key)))
}
