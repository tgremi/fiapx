package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/fiapx/processing-worker/internal/domain"
	amqp091 "github.com/rabbitmq/amqp091-go"
)

type fakeChannel struct {
	mu         sync.Mutex
	exchange   string
	key        string
	msg        amqp091.Publishing
	publishErr error
}

func (f *fakeChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp091.Table) error {
	return nil
}

func (f *fakeChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp091.Table) (amqp091.Queue, error) {
	return amqp091.Queue{}, nil
}

func (f *fakeChannel) QueueBind(name, key, exchange string, noWait bool, args amqp091.Table) error {
	return nil
}

func (f *fakeChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp091.Publishing) error {
	if f.publishErr != nil {
		return f.publishErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exchange = exchange
	f.key = key
	f.msg = msg
	return nil
}

func (f *fakeChannel) Close() error { return nil }

func TestPublishVideoFailedUnit(t *testing.T) {
	ch := &fakeChannel{}
	p := &Publisher{ch: ch}

	job := domain.ProcessingJob{VideoID: "vid-1", UserID: "u-1", OriginalKey: "u-1/vid-1.mp4"}
	if err := p.PublishVideoFailed(context.Background(), job, "ffmpeg falhou"); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if ch.exchange != exchangeName {
		t.Fatalf("exchange = %q, want %q", ch.exchange, exchangeName)
	}
	if ch.key != routingKeyFailed {
		t.Fatalf("routing key = %q, want %q", ch.key, routingKeyFailed)
	}

	var ev videoFailedEvent
	if err := json.Unmarshal(ch.msg.Body, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.VideoID != "vid-1" || ev.UserID != "u-1" || ev.Error != "ffmpeg falhou" {
		t.Fatalf("evento inesperado: %+v", ev)
	}
}

func TestPublishVideoFailedPropagatesError(t *testing.T) {
	ch := &fakeChannel{publishErr: errors.New("conn closed")}
	p := &Publisher{ch: ch}

	err := p.PublishVideoFailed(context.Background(), domain.ProcessingJob{VideoID: "v"}, "boom")
	if err == nil {
		t.Fatal("esperava erro de publicação")
	}
}
