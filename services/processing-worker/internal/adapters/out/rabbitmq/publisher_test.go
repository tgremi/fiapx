//go:build integration

package rabbitmq

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fiapx/processing-worker/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

func TestPublishVideoFailed(t *testing.T) {
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

	p, err := NewPublisher(url)
	if err != nil {
		t.Fatalf("publisher: %v", err)
	}
	defer p.Close()

	job := domain.ProcessingJob{VideoID: "vid-1", UserID: "u-1", OriginalKey: "u-1/vid-1.mp4"}
	if err := p.PublishVideoFailed(ctx, job, "ffmpeg falhou"); err != nil {
		t.Fatalf("publish: %v", err)
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer func() { _ = ch.Close() }()

	if _, err := ch.QueueDeclare(notificationQueue, true, false, false, false, nil); err != nil {
		t.Fatalf("queue declare: %v", err)
	}
	msgs, err := ch.Consume(notificationQueue, "", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}

	select {
	case msg := <-msgs:
		var ev videoFailedEvent
		if err := json.Unmarshal(msg.Body, &ev); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if ev.VideoID != "vid-1" || ev.Error != "ffmpeg falhou" {
			t.Fatalf("evento inesperado: %+v", ev)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout esperando mensagem no broker")
	}
}
