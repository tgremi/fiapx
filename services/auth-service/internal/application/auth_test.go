package application

import (
	"context"
	"testing"

	"github.com/fiapx/auth-service/internal/domain"
)

type fakeUserRepo struct {
	users map[string]domain.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[string]domain.User{}}
}

func (f *fakeUserRepo) Create(ctx context.Context, email, passwordHash string) (domain.User, error) {
	if _, ok := f.users[email]; ok {
		return domain.User{}, domain.ErrUserAlreadyExists
	}
	u := domain.User{ID: "id-" + email, Email: email, PasswordHash: passwordHash}
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

func (fakeHasher) Hash(password string) (string, error) { return "hashed-" + password, nil }
func (fakeHasher) Compare(hash, password string) bool   { return hash == "hashed-"+password }

type fakeTokens struct{}

func (fakeTokens) Generate(userID, email string) (string, error) { return "token-" + userID, nil }
func (fakeTokens) Validate(token string) (domain.Claims, error)  { return domain.Claims{}, nil }

func TestRegisterAndLogin(t *testing.T) {
	uc := NewAuthUseCase(newFakeUserRepo(), fakeHasher{}, fakeTokens{})
	ctx := context.Background()

	if _, err := uc.Register(ctx, "a@b.com", "secret123"); err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := uc.Register(ctx, "a@b.com", "secret123"); err != domain.ErrUserAlreadyExists {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}

	if _, err := uc.Register(ctx, "x@b.com", "123"); err != domain.ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	tok, err := uc.Login(ctx, "a@b.com", "secret123")
	if err != nil || tok != "token-id-a@b.com" {
		t.Fatalf("login: tok=%q err=%v", tok, err)
	}

	if _, err := uc.Login(ctx, "a@b.com", "wrong"); err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
