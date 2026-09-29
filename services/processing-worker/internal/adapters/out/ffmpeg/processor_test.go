package ffmpeg

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg não disponível no PATH")
	}
}

func TestProcessRealVideo(t *testing.T) {
	requireFFmpeg(t)

	dir := t.TempDir()
	input := filepath.Join(dir, "input.mp4")

	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", "testsrc=duration=2:size=128x128:rate=10",
		"-pix_fmt", "yuv420p", "-y", input,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gerar vídeo de teste: %v: %s", err, out)
	}

	f, err := os.Open(input)
	if err != nil {
		t.Fatalf("abrir input: %v", err)
	}
	defer func() { _ = f.Close() }()

	p := NewProcessor()
	r, err := p.Process(context.Background(), f)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	defer func() { _ = r.Close() }()

	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ler zip: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("abrir zip resultante: %v", err)
	}
	if len(zr.File) == 0 {
		t.Fatal("zip não contém frames")
	}
	for _, zf := range zr.File {
		if filepath.Ext(zf.Name) != ".png" {
			t.Fatalf("arquivo inesperado no zip: %s", zf.Name)
		}
	}
}

func TestProcessInvalidInput(t *testing.T) {
	requireFFmpeg(t)

	p := NewProcessor()
	if _, err := p.Process(context.Background(), bytes.NewReader([]byte("isto não é um vídeo"))); err == nil {
		t.Fatal("esperado erro ao processar entrada inválida")
	}
}
