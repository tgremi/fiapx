package application

import (
	"context"
	"errors"
	"testing"

	"github.com/fiapx/notification-service/internal/domain"
)

type fakeNotifier struct {
	sent []domain.Notification
	err  error
}

func (f *fakeNotifier) Send(ctx context.Context, n domain.Notification) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, n)
	return nil
}

type fakeUserRepo struct {
	user domain.User
	err  error
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id string) (domain.User, error) {
	if f.err != nil {
		return domain.User{}, f.err
	}
	return f.user, nil
}

func TestNotifyErrorSuccess(t *testing.T) {
	notifier := &fakeNotifier{}
	repo := &fakeUserRepo{user: domain.User{ID: "u1", Email: "u1@fiapx.com"}}
	uc := NewNotificationUseCase(notifier, repo)

	err := uc.NotifyError(context.Background(), "u1", "v1", "ffmpeg bug")
	if err != nil {
		t.Fatalf("notify: %v", err)
	}

	if len(notifier.sent) != 1 {
		t.Fatalf("e-mails enviados = %d", len(notifier.sent))
	}
	if notifier.sent[0].Email != "u1@fiapx.com" {
		t.Fatalf("destinatário = %s", notifier.sent[0].Email)
	}
}

func TestNotifyErrorUserNotFound(t *testing.T) {
	notifier := &fakeNotifier{}
	repo := &fakeUserRepo{err: domain.ErrUserNotFound}
	uc := NewNotificationUseCase(notifier, repo)

	if err := uc.NotifyError(context.Background(), "u1", "v1", "x"); err != domain.ErrUserNotFound {
		t.Fatalf("esperado ErrUserNotFound, got %v", err)
	}
	if len(notifier.sent) != 0 {
		t.Fatalf("não deveria enviar e-mail, enviados = %d", len(notifier.sent))
	}
}

func TestNotifyErrorSendFailure(t *testing.T) {
	notifier := &fakeNotifier{err: errors.New("smtp down")}
	repo := &fakeUserRepo{user: domain.User{ID: "u1", Email: "u1@fiapx.com"}}
	uc := NewNotificationUseCase(notifier, repo)

	if err := uc.NotifyError(context.Background(), "u1", "v1", "x"); err == nil {
		t.Fatal("esperado erro ao enviar e-mail")
	}
}
