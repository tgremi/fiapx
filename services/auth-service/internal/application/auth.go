package application

import (
	"context"
	"strings"

	"github.com/fiapx/auth-service/internal/domain"
)

type AuthUseCase struct {
	users  domain.UserRepository
	hasher domain.PasswordHasher
	tokens domain.TokenService
}

func NewAuthUseCase(users domain.UserRepository, hasher domain.PasswordHasher, tokens domain.TokenService) *AuthUseCase {
	return &AuthUseCase{users: users, hasher: hasher, tokens: tokens}
}

func (uc *AuthUseCase) Register(ctx context.Context, email, password string) (domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(password) < 6 {
		return domain.User{}, domain.ErrInvalidInput
	}

	hash, err := uc.hasher.Hash(password)
	if err != nil {
		return domain.User{}, err
	}

	return uc.users.Create(ctx, email, hash)
}

func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		return "", domain.ErrInvalidCredentials
	}

	if !uc.hasher.Compare(user.PasswordHash, password) {
		return "", domain.ErrInvalidCredentials
	}

	return uc.tokens.Generate(user.ID, user.Email)
}
