package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/fiapx/notification-service/internal/application"
	"github.com/fiapx/notification-service/internal/domain"
	amqp091 "github.com/rabbitmq/amqp091-go"
)

type fakeChannel struct {
	mu         sync.Mutex
	published  []fakePublish
	deliveries chan amqp091.Delivery
	publishErr error
}

type fakePublish struct {
	queue string
	msg   amqp091.Publishing
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

func (f *fakeChannel) Qos(prefetchCount, prefetchSize int, global bool) error { return nil }

func (f *fakeChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp091.Table) (<-chan amqp091.Delivery, error) {
	return f.deliveries, nil
}

func (f *fakeChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp091.Publishing) error {
	if f.publishErr != nil {
		return f.publishErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.published = append(f.published, fakePublish{queue: key, msg: msg})
	return nil
}

func (f *fakeChannel) Close() error { return nil }

type fakeAck struct {
	mu          sync.Mutex
	acked       int
	nacked      int
	lastRequeue bool
}

func (a *fakeAck) Ack(tag uint64, multiple bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.acked++
	return nil
}

func (a *fakeAck) Nack(tag uint64, multiple, requeue bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nacked++
	a.lastRequeue = requeue
	return nil
}

func (a *fakeAck) Reject(tag uint64, requeue bool) error { return nil }

type stubNotifier struct {
	err error
}

func (s stubNotifier) Send(ctx context.Context, n domain.Notification) error { return s.err }

type stubUsers struct {
	user domain.User
}

func (s stubUsers) FindByID(ctx context.Context, id string) (domain.User, error) {
	return s.user, nil
}

func eventually(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condição não satisfeita a tempo: %s", msg)
}

func newConsumerForTest(ch *fakeChannel, notifier domain.Notifier) *Consumer {
	uc := application.NewNotificationUseCase(notifier, stubUsers{user: domain.User{ID: "u-1", Email: "a@b.com"}})
	return &Consumer{ch: ch, usecase: uc}
}

func runConsumer(t *testing.T, c *Consumer) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = c.Run(ctx) }()
	return cancel
}

func TestRunNotifiesAndAcks(t *testing.T) {
	ack := &fakeAck{}
	ch := &fakeChannel{deliveries: make(chan amqp091.Delivery, 1)}
	c := newConsumerForTest(ch, stubNotifier{})

	body, _ := json.Marshal(videoFailedEvent{VideoID: "vid-1", UserID: "u-1", Error: "boom"})
	ch.deliveries <- amqp091.Delivery{Acknowledger: ack, Body: body}

	cancel := runConsumer(t, c)
	defer cancel()

	eventually(t, func() bool { ack.mu.Lock(); defer ack.mu.Unlock(); return ack.acked == 1 }, "esperava 1 ack")
}

func TestRunUnmarshalErrorNacksWithoutRequeue(t *testing.T) {
	ack := &fakeAck{}
	ch := &fakeChannel{deliveries: make(chan amqp091.Delivery, 1)}
	c := newConsumerForTest(ch, stubNotifier{})

	ch.deliveries <- amqp091.Delivery{Acknowledger: ack, Body: []byte("nao-e-json")}

	cancel := runConsumer(t, c)
	defer cancel()

	eventually(t, func() bool {
		ack.mu.Lock()
		defer ack.mu.Unlock()
		return ack.nacked == 1 && !ack.lastRequeue
	}, "esperava 1 nack sem requeue")
}

func TestRunNotifyErrorPublishesToRetry(t *testing.T) {
	ack := &fakeAck{}
	ch := &fakeChannel{deliveries: make(chan amqp091.Delivery, 1)}
	c := newConsumerForTest(ch, stubNotifier{err: errors.New("smtp down")})

	body, _ := json.Marshal(videoFailedEvent{VideoID: "vid-1", UserID: "u-1", Error: "boom"})
	ch.deliveries <- amqp091.Delivery{Acknowledger: ack, Body: body}

	cancel := runConsumer(t, c)
	defer cancel()

	eventually(t, func() bool {
		ch.mu.Lock()
		defer ch.mu.Unlock()
		return len(ch.published) == 1
	}, "esperava 1 publicação")

	ch.mu.Lock()
	p := ch.published[0]
	ch.mu.Unlock()

	if p.queue != retryQueueName {
		t.Fatalf("fila = %q, want %q", p.queue, retryQueueName)
	}
	if got := getRetryCount(p.msg.Headers); got != 1 {
		t.Fatalf("x-retry-count = %d, want 1", got)
	}
}

func TestRunNotifyErrorDeadLettersAfterMaxRetries(t *testing.T) {
	ack := &fakeAck{}
	ch := &fakeChannel{deliveries: make(chan amqp091.Delivery, 1)}
	c := newConsumerForTest(ch, stubNotifier{err: errors.New("smtp down")})

	body, _ := json.Marshal(videoFailedEvent{VideoID: "vid-1", UserID: "u-1", Error: "boom"})
	ch.deliveries <- amqp091.Delivery{
		Acknowledger: ack,
		Body:         body,
		Headers:      amqp091.Table{retryHeader: int32(maxRetries)},
	}

	cancel := runConsumer(t, c)
	defer cancel()

	eventually(t, func() bool {
		ch.mu.Lock()
		defer ch.mu.Unlock()
		return len(ch.published) == 1
	}, "esperava 1 publicação na DLQ")

	ch.mu.Lock()
	p := ch.published[0]
	ch.mu.Unlock()

	if p.queue != dlqQueueName {
		t.Fatalf("fila = %q, want %q", p.queue, dlqQueueName)
	}
}

func TestRunNotifyErrorNacksWhenPublishFails(t *testing.T) {
	ack := &fakeAck{}
	ch := &fakeChannel{
		deliveries: make(chan amqp091.Delivery, 1),
		publishErr: errors.New("channel closed"),
	}
	c := newConsumerForTest(ch, stubNotifier{err: errors.New("smtp down")})

	body, _ := json.Marshal(videoFailedEvent{VideoID: "vid-1", UserID: "u-1", Error: "boom"})
	ch.deliveries <- amqp091.Delivery{Acknowledger: ack, Body: body}

	cancel := runConsumer(t, c)
	defer cancel()

	eventually(t, func() bool {
		ack.mu.Lock()
		defer ack.mu.Unlock()
		return ack.nacked == 1 && ack.lastRequeue
	}, "esperava nack com requeue quando a publicação falha")
}

func TestRunStopsOnContextCancel(t *testing.T) {
	ch := &fakeChannel{deliveries: make(chan amqp091.Delivery)}
	c := newConsumerForTest(ch, stubNotifier{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = c.Run(ctx)
		close(done)
	}()
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run não parou após cancelamento do contexto")
	}
}
