//go:build integration

package amqp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fiapx/notification-service/internal/application"
	"github.com/fiapx/notification-service/internal/domain"
	amqp091 "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

type fakeNotifier struct {
	notified chan domain.Notification
}

func (f *fakeNotifier) Send(ctx context.Context, n domain.Notification) error {
	f.notified <- n
	return nil
}

type fakeUsers struct {
	user domain.User
}

func (f fakeUsers) FindByID(ctx context.Context, id string) (domain.User, error) {
	return f.user, nil
}

func TestConsumerNotifiesAndAcks(t *testing.T) {
	ctx := context.Background()

	c, err := rabbitmq.Run(ctx, "rabbitmq:3-management-alpine",
		rabbitmq.WithAdminUsername("guest"),
		rabbitmq.WithAdminPassword("guest"),
	)
	if err != nil {
		t.Fatalf("rabbitmq container: %v", err)
	}
	t.Cleanup(func() { _ = c.Terminate(ctx) })

	host, _ := c.Host(ctx)
	port, _ := c.MappedPort(ctx, "5672")
	url := "amqp://guest:guest@" + host + ":" + port.Port() + "/"

	notifier := &fakeNotifier{notified: make(chan domain.Notification, 1)}
	uc := application.NewNotificationUseCase(notifier, fakeUsers{user: domain.User{ID: "u-1", Email: "a@b.com"}})

	consumer, err := NewConsumer(url, uc)
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	defer consumer.Close()

	// publica o evento de falha no exchange/tópico consumido pelo serviço
	conn, err := amqp091.Dial(url)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer func() { _ = ch.Close() }()

	body, _ := json.Marshal(videoFailedEvent{VideoID: "vid-1", UserID: "u-1", Error: "boom"})
	if err := ch.Publish(exchangeName, routingKeyFailed, false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { _ = consumer.Run(runCtx) }()

	select {
	case n := <-notifier.notified:
		if n.Email != "a@b.com" {
			t.Fatalf("email inesperado: %q", n.Email)
		}
		if n.Subject == "" || n.Body == "" {
			t.Fatalf("notificação vazia: %+v", n)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("timeout esperando notificação")
	}
}
