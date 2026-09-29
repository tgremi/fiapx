package domain

import (
	"context"
	"errors"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
}

type Claims struct {
	UserID string
	Email  string
}

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

type TokenService interface {
	Generate(userID, email string) (string, error)
	Validate(token string) (Claims, error)
}
