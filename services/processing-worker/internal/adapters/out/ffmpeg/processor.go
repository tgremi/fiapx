package ffmpeg

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type Processor struct{}

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) Process(ctx context.Context, src io.Reader) (io.ReadCloser, error) {
	dir, err := os.MkdirTemp("", "fiapx-")
	if err != nil {
		return nil, err
	}

	input := filepath.Join(dir, "input")
	f, err := os.Create(input)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	if _, err := io.Copy(f, src); err != nil {
		_ = f.Close()
		_ = os.RemoveAll(dir)
		return nil, err
	}
	_ = f.Close()

	framePattern := filepath.Join(dir, "frame_%04d.png")
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", input, "-vf", "fps=1", "-y", framePattern)
	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("ffmpeg falhou: %s", string(out))
	}

	frames, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil || len(frames) == 0 {
		_ = os.RemoveAll(dir)
		return nil, errors.New("nenhum frame extraído do vídeo")
	}

	zipPath := filepath.Join(dir, "frames.zip")
	if err := zipFrames(frames, zipPath); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}

	zf, err := os.Open(zipPath)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}

	return &tempZip{File: zf, dir: dir}, nil
}

type tempZip struct {
	*os.File
	dir string
}

func (t *tempZip) Close() error {
	err := t.File.Close()
	_ = os.RemoveAll(t.dir)
	return err
}

func zipFrames(files []string, dest string) error {
	zf, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = zf.Close() }()

	w := zip.NewWriter(zf)
	defer func() { _ = w.Close() }()

	for _, file := range files {
		if err := addToZip(w, file); err != nil {
			return err
		}
	}
	return nil
}

func addToZip(w *zip.Writer, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = filepath.Base(filename)
	hdr.Method = zip.Deflate

	writer, err := w.CreateHeader(hdr)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, f)
	return err
}
