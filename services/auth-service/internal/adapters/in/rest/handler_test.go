package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fiapx/auth-service/internal/application"
	"github.com/fiapx/auth-service/internal/domain"
	"github.com/gin-gonic/gin"
)

type fakeUserRepo struct {
	users map[string]domain.User
}

func newFakeUserRepo() *fakeUserRepo { return &fakeUserRepo{users: map[string]domain.User{}} }

func (f *fakeUserRepo) Create(ctx context.Context, email, hash string) (domain.User, error) {
	if _, ok := f.users[email]; ok {
		return domain.User{}, domain.ErrUserAlreadyExists
	}
	u := domain.User{ID: "id-" + email, Email: email, PasswordHash: hash}
	f.users[email] = u
	return u, nil
}

func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := f.users[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return &u, nil
}

type fakeHasher struct{}

func (fakeHasher) Hash(p string) (string, error) { return "hashed-" + p, nil }
func (fakeHasher) Compare(h, p string) bool       { return h == "hashed-"+p }

type fakeTokens struct{}

func (fakeTokens) Generate(userID, email string) (string, error) { return "tok-" + userID, nil }
func (fakeTokens) Validate(token string) (domain.Claims, error)  { return domain.Claims{}, nil }

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	uc := application.NewAuthUseCase(newFakeUserRepo(), fakeHasher{}, fakeTokens{})
	return NewHandler(uc).Router()
}

func doJSON(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHealth(t *testing.T) {
	w := doJSON(newTestRouter(), "GET", "/health", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("health = %d", w.Code)
	}
}

func TestRegisterHTTP(t *testing.T) {
	r := newTestRouter()

	w := doJSON(r, "POST", "/register", map[string]string{"email": "a@b.com", "password": "secret1"})
	if w.Code != http.StatusCreated {
		t.Fatalf("register = %d", w.Code)
	}

	w = doJSON(r, "POST", "/register", map[string]string{"email": "a@b.com", "password": "secret1"})
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicado = %d", w.Code)
	}

	w = doJSON(r, "POST", "/register", map[string]string{"email": "x@b.com", "password": "123"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("senha curta = %d", w.Code)
	}

	w = doJSON(r, "POST", "/register", "payload-invalido")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("payload inválido = %d", w.Code)
	}
}

func TestLoginHTTP(t *testing.T) {
	r := newTestRouter()

	_ = doJSON(r, "POST", "/register", map[string]string{"email": "a@b.com", "password": "secret1"})

	w := doJSON(r, "POST", "/login", map[string]string{"email": "a@b.com", "password": "secret1"})
	if w.Code != http.StatusOK {
		t.Fatalf("login = %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp["token"] == "" {
		t.Fatalf("token ausente: %v %s", err, w.Body.String())
	}

	w = doJSON(r, "POST", "/login", map[string]string{"email": "a@b.com", "password": "errada"})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("senha errada = %d", w.Code)
	}
}
