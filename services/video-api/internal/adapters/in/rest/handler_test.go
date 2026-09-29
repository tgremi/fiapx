package rest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fiapx/video-api/internal/application"
	"github.com/fiapx/video-api/internal/domain"
	"github.com/gin-gonic/gin"
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

type fakeCache struct{}

func (fakeCache) Get(ctx context.Context, key string) ([]byte, bool, error) { return nil, false, nil }
func (fakeCache) Set(ctx context.Context, key string, v []byte, ttl time.Duration) error {
	return nil
}
func (fakeCache) Del(ctx context.Context, key string) error { return nil }

type fakePublisher struct{}

func (fakePublisher) PublishVideoUploaded(ctx context.Context, v domain.Video) error { return nil }

type fakeValidator struct {
	claims domain.Claims
	err    error
}

func (f fakeValidator) Validate(token string) (domain.Claims, error) { return f.claims, f.err }

func newTestRouter(validator domain.TokenValidator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	uc := application.NewVideoUseCase(newFakeRepo(), newFakeStorage(), fakeCache{}, fakePublisher{})
	return NewHandler(uc, validator).Router()
}

func seedVideo(repo *fakeRepo, storage *fakeStorage) {
	repo.videos["vid-1"] = domain.Video{
		ID: "vid-1", UserID: "user-1", Status: domain.StatusCompleted,
		OriginalKey: "user-1/vid-1.mp4", ZipKey: "user-1/vid-1.zip", CreatedAt: time.Now(),
	}
	storage.data["user-1/vid-1.zip"] = []byte("zip-content")
}

func TestHealth(t *testing.T) {
	r := newTestRouter(fakeValidator{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("health = %d", w.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	r := newTestRouter(fakeValidator{err: errors.New("invalid")})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/videos", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sem token = %d", w.Code)
	}

	req := httptest.NewRequest("GET", "/videos", nil)
	req.Header.Set("Authorization", "Bearer invalido")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("token inválido = %d", w.Code)
	}
}

func TestListVideos(t *testing.T) {
	repo := newFakeRepo()
	storage := newFakeStorage()
	seedVideo(repo, storage)

	uc := application.NewVideoUseCase(repo, storage, fakeCache{}, fakePublisher{})
	r := NewHandler(uc, fakeValidator{claims: domain.Claims{UserID: "user-1"}}).Router()

	req := httptest.NewRequest("GET", "/videos", nil)
	req.Header.Set("Authorization", "Bearer ok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list = %d: %s", w.Code, w.Body.String())
	}
}

func TestGetVideo(t *testing.T) {
	repo := newFakeRepo()
	storage := newFakeStorage()
	seedVideo(repo, storage)

	uc := application.NewVideoUseCase(repo, storage, fakeCache{}, fakePublisher{})
	r := NewHandler(uc, fakeValidator{claims: domain.Claims{UserID: "user-1"}}).Router()

	req := httptest.NewRequest("GET", "/videos/vid-1", nil)
	req.Header.Set("Authorization", "Bearer ok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get = %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/videos/nao-existe", nil)
	req.Header.Set("Authorization", "Bearer ok")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get inexistente = %d", w.Code)
	}
}

func TestDownloadVideo(t *testing.T) {
	repo := newFakeRepo()
	storage := newFakeStorage()
	seedVideo(repo, storage)

	uc := application.NewVideoUseCase(repo, storage, fakeCache{}, fakePublisher{})
	r := NewHandler(uc, fakeValidator{claims: domain.Claims{UserID: "user-1"}}).Router()

	req := httptest.NewRequest("GET", "/videos/vid-1/download", nil)
	req.Header.Set("Authorization", "Bearer ok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("download = %d", w.Code)
	}
	if w.Body.String() != "zip-content" {
		t.Fatalf("conteúdo = %q", w.Body.String())
	}
}

func TestUploadVideo(t *testing.T) {
	uc := application.NewVideoUseCase(newFakeRepo(), newFakeStorage(), fakeCache{}, fakePublisher{})
	r := NewHandler(uc, fakeValidator{claims: domain.Claims{UserID: "user-1"}}).Router()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("video", "video.mp4")
	_, _ = fw.Write([]byte("video-bytes"))
	_ = mw.Close()

	req := httptest.NewRequest("POST", "/videos", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer ok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("upload = %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadMissingFile(t *testing.T) {
	uc := application.NewVideoUseCase(newFakeRepo(), newFakeStorage(), fakeCache{}, fakePublisher{})
	r := NewHandler(uc, fakeValidator{claims: domain.Claims{UserID: "user-1"}}).Router()

	req := httptest.NewRequest("POST", "/videos", bytes.NewReader(nil))
	req.Header.Set("Authorization", "Bearer ok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("sem arquivo = %d", w.Code)
	}
}
