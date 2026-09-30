package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/fiapx/video-api/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type fakeChannel struct {
	mu         sync.Mutex
	exchange   string
	key        string
	msg        amqp.Publishing
	publishErr error
}

func (f *fakeChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	return nil
}

func (f *fakeChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{}, nil
}

func (f *fakeChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return nil
}

func (f *fakeChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
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

func TestPublishVideoUploadedUnit(t *testing.T) {
	ch := &fakeChannel{}
	p := &Publisher{ch: ch}

	v := domain.Video{ID: "vid-1", UserID: "u-1", OriginalKey: "u-1/vid-1.mp4"}
	if err := p.PublishVideoUploaded(context.Background(), v); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if ch.exchange != exchangeName {
		t.Fatalf("exchange = %q, want %q", ch.exchange, exchangeName)
	}
	if ch.key != routingKeyUploaded {
		t.Fatalf("routing key = %q, want %q", ch.key, routingKeyUploaded)
	}

	var ev videoUploadedEvent
	if err := json.Unmarshal(ch.msg.Body, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.VideoID != "vid-1" || ev.UserID != "u-1" || ev.OriginalKey != "u-1/vid-1.mp4" {
		t.Fatalf("evento inesperado: %+v", ev)
	}
}

func TestPublishVideoUploadedPropagatesError(t *testing.T) {
	ch := &fakeChannel{publishErr: errors.New("conn closed")}
	p := &Publisher{ch: ch}

	err := p.PublishVideoUploaded(context.Background(), domain.Video{ID: "v"})
	if err == nil {
		t.Fatal("esperava erro de publicação")
	}
}
