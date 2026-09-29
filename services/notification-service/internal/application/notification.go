package application

import (
	"context"
	"fmt"

	"github.com/fiapx/notification-service/internal/domain"
	"github.com/fiapx/notification-service/internal/metrics"
)

type NotificationUseCase struct {
	notifier domain.Notifier
	users    domain.UserRepository
}

func NewNotificationUseCase(notifier domain.Notifier, users domain.UserRepository) *NotificationUseCase {
	return &NotificationUseCase{notifier: notifier, users: users}
}

func (uc *NotificationUseCase) NotifyError(ctx context.Context, userID, videoID, message string) error {
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	n := domain.Notification{
		Email:   user.Email,
		Subject: fmt.Sprintf("FIAP X — falha no processamento do vídeo %s", videoID),
		Body:    fmt.Sprintf("Olá,\n\nO processamento do seu vídeo (%s) falhou.\n\nMotivo: %s\n\nEquipe FIAP X", videoID, message),
	}

	if err := uc.notifier.Send(ctx, n); err != nil {
		metrics.EmailsFailedInc()
		return err
	}

	metrics.EmailsSentInc()
	return nil
}
