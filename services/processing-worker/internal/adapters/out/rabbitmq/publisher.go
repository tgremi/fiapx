package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fiapx/processing-worker/internal/domain"
	amqp091 "github.com/rabbitmq/amqp091-go"
)

const (
	exchangeName      = "video.events"
	routingKeyFailed  = "video.failed"
	notificationQueue = "notification"
)

type videoFailedEvent struct {
	VideoID string `json:"video_id"`
	UserID  string `json:"user_id"`
	Error   string `json:"error"`
}

type amqpChannel interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp091.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp091.Table) (amqp091.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp091.Table) error
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp091.Publishing) error
	Close() error
}

type Publisher struct {
	conn *amqp091.Connection
	ch   amqpChannel
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}
	if _, err := ch.QueueDeclare(notificationQueue, true, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := ch.QueueBind(notificationQueue, routingKeyFailed, exchangeName, false, nil); err != nil {
		return nil, err
	}

	return &Publisher{conn: conn, ch: ch}, nil
}

func (p *Publisher) PublishVideoFailed(ctx context.Context, job domain.ProcessingJob, errMsg string) error {
	body, err := json.Marshal(videoFailedEvent{
		VideoID: job.VideoID,
		UserID:  job.UserID,
		Error:   errMsg,
	})
	if err != nil {
		return err
	}

	return p.ch.PublishWithContext(ctx, exchangeName, routingKeyFailed, false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
		Timestamp:   time.Now(),
	})
}

func (p *Publisher) Close() {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}
