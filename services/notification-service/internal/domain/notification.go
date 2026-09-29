package domain

import (
	"context"
	"errors"
)

var ErrUserNotFound = errors.New("user not found")

type User struct {
	ID    string
	Email string
}

type Notification struct {
	Email   string
	Subject string
	Body    string
}

type Notifier interface {
	Send(ctx context.Context, n Notification) error
}

type UserRepository interface {
	FindByID(ctx context.Context, id string) (User, error)
}
