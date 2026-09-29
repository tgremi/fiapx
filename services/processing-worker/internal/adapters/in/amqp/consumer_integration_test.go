//go:build integration

package amqp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/fiapx/processing-worker/internal/application"
	"github.com/fiapx/processing-worker/internal/domain"
	amqp091 "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

type fakeProcessor struct {
	out io.ReadCloser
}

func (f fakeProcessor) Process(ctx context.Context, src io.Reader) (io.ReadCloser, error) {
	return f.out, nil
}

type fakeStorage struct{}

func (fakeStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("video"))), nil
}
func (fakeStorage) Upload(ctx context.Context, key string, data io.Reader) error {
	_, _ = io.Copy(io.Discard, data)
	return nil
}

type fakeRepo struct {
	status  domain.VideoStatus
	updated chan domain.VideoStatus
}

func (f *fakeRepo) UpdateStatus(ctx context.Context, id string, s domain.VideoStatus, zip, msg string) error {
	if f.updated != nil {
		f.updated <- s
	}
	return nil
}
func (f *fakeRepo) GetStatus(ctx context.Context, id string) (domain.VideoStatus, error) {
	return f.status, nil
}

type fakePublisher struct{}

func (fakePublisher) PublishVideoFailed(ctx context.Context, job domain.ProcessingJob, errMsg string) error {
	return nil
}

func TestConsumerProcessesMessage(t *testing.T) {
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

	repo := &fakeRepo{status: domain.StatusPending, updated: make(chan domain.VideoStatus, 1)}
	uc := application.NewProcessingUseCase(
		fakeProcessor{out: io.NopCloser(bytes.NewReader([]byte("zip")))},
		fakeStorage{},
		repo,
		fakePublisher{},
	)

	consumer, err := NewConsumer(url, uc)
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	defer consumer.Close()

	// publica o evento de upload no tópico consumido pelo worker
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

	body, _ := json.Marshal(videoUploadedEvent{VideoID: "vid-1", UserID: "u-1", OriginalKey: "u-1/vid-1.mp4"})
	if err := ch.Publish(exchangeName, routingKeyUploaded, false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { _ = consumer.Run(runCtx) }()

	deadline := time.After(15 * time.Second)
	for {
		select {
		case s := <-repo.updated:
			if s == domain.StatusCompleted {
				return
			}
		case <-deadline:
			t.Fatal("timeout esperando processamento concluído")
		}
	}
}
