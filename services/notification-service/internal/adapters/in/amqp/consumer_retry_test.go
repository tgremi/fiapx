//go:build integration

package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/fiapx/notification-service/internal/application"
	"github.com/fiapx/notification-service/internal/domain"
	amqp091 "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

type failingNotifier struct{}

func (failingNotifier) Send(ctx context.Context, n domain.Notification) error {
	return errors.New("smtp down")
}

func startBroker(t *testing.T) (url string) {
	t.Helper()
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
	return "amqp://guest:guest@" + host + ":" + port.Port() + "/"
}

func startQueueConsumer(t *testing.T, url, queue string) <-chan amqp091.Delivery {
	t.Helper()
	conn, err := amqp091.Dial(url)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	t.Cleanup(func() { _ = ch.Close() })

	msgs, err := ch.Consume(queue, "", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume %s: %v", queue, err)
	}
	return msgs
}

func publishFailed(t *testing.T, url string, headers amqp091.Table) {
	t.Helper()
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
		Headers:     headers,
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}
}

func TestConsumerRetriesOnFailure(t *testing.T) {
	url := startBroker(t)

	uc := application.NewNotificationUseCase(failingNotifier{}, fakeUsers{user: domain.User{ID: "u-1", Email: "a@b.com"}})
	consumer, err := NewConsumer(url, uc)
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	defer consumer.Close()

	retryMsgs := startQueueConsumer(t, url, retryQueueName)

	publishFailed(t, url, nil)

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = consumer.Run(runCtx) }()

	select {
	case msg := <-retryMsgs:
		if got := getRetryCount(msg.Headers); got != 1 {
			t.Fatalf("x-retry-count = %d, want 1", got)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout esperando mensagem na fila de retry")
	}
}

func TestConsumerDeadLettersAfterMaxRetries(t *testing.T) {
	url := startBroker(t)

	uc := application.NewNotificationUseCase(failingNotifier{}, fakeUsers{user: domain.User{ID: "u-1", Email: "a@b.com"}})
	consumer, err := NewConsumer(url, uc)
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	defer consumer.Close()

	dlqMsgs := startQueueConsumer(t, url, dlqQueueName)

	publishFailed(t, url, amqp091.Table{retryHeader: int32(maxRetries)})

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = consumer.Run(runCtx) }()

	select {
	case msg := <-dlqMsgs:
		if got := getRetryCount(msg.Headers); got != maxRetries {
			t.Fatalf("x-retry-count = %d, want %d", got, maxRetries)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout esperando mensagem na DLQ")
	}
}
