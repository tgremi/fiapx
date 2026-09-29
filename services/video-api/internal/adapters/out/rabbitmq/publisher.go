package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fiapx/video-api/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	exchangeName        = "video.events"
	routingKeyUploaded  = "video.uploaded"
	processingQueueName = "video.processing"
)

type videoUploadedEvent struct {
	VideoID     string `json:"video_id"`
	UserID      string `json:"user_id"`
	OriginalKey string `json:"original_key"`
}

type Publisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
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
	if _, err := ch.QueueDeclare(processingQueueName, true, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := ch.QueueBind(processingQueueName, routingKeyUploaded, exchangeName, false, nil); err != nil {
		return nil, err
	}

	return &Publisher{conn: conn, ch: ch}, nil
}

func (p *Publisher) PublishVideoUploaded(ctx context.Context, v domain.Video) error {
	body, err := json.Marshal(videoUploadedEvent{
		VideoID:     v.ID,
		UserID:      v.UserID,
		OriginalKey: v.OriginalKey,
	})
	if err != nil {
		return err
	}

	return p.ch.PublishWithContext(ctx, exchangeName, routingKeyUploaded, false, false, amqp.Publishing{
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
