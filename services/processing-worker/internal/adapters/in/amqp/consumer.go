package amqp

import (
	"context"
	"encoding/json"
	"log"

	"github.com/fiapx/processing-worker/internal/application"
	"github.com/fiapx/processing-worker/internal/domain"
	amqp091 "github.com/rabbitmq/amqp091-go"
)

const (
	exchangeName       = "video.events"
	routingKeyUploaded = "video.uploaded"
	queueName          = "video.processing"
	retryQueueName     = "video.processing.retry"
	dlqQueueName       = "video.processing.dlq"
	retryTTLMs         = 5000
	maxRetries         = 3
	retryHeader        = "x-retry-count"
)

type videoUploadedEvent struct {
	VideoID     string `json:"video_id"`
	UserID      string `json:"user_id"`
	OriginalKey string `json:"original_key"`
}

type amqpChannel interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp091.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp091.Table) (amqp091.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp091.Table) error
	Qos(prefetchCount, prefetchSize int, global bool) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp091.Table) (<-chan amqp091.Delivery, error)
	Publish(exchange, key string, mandatory, immediate bool, msg amqp091.Publishing) error
	Close() error
}

type Consumer struct {
	conn    *amqp091.Connection
	ch      amqpChannel
	usecase *application.ProcessingUseCase
}

func NewConsumer(url string, usecase *application.ProcessingUseCase) (*Consumer, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := declareTopology(ch); err != nil {
		return nil, err
	}
	if err := ch.Qos(1, 0, false); err != nil {
		return nil, err
	}

	return &Consumer{conn: conn, ch: ch, usecase: usecase}, nil
}

func declareTopology(ch amqpChannel) error {
	if err := ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(queueName, routingKeyUploaded, exchangeName, false, nil); err != nil {
		return err
	}

	retryArgs := amqp091.Table{
		"x-message-ttl":             int32(retryTTLMs),
		"x-dead-letter-exchange":    exchangeName,
		"x-dead-letter-routing-key": routingKeyUploaded,
	}
	if _, err := ch.QueueDeclare(retryQueueName, true, false, false, false, retryArgs); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(dlqQueueName, true, false, false, false, nil); err != nil {
		return err
	}

	return nil
}

func (c *Consumer) Run(ctx context.Context) error {
	msgs, err := c.ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}

			var event videoUploadedEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				_ = msg.Nack(false, false)
				continue
			}

			job := domain.ProcessingJob{
				VideoID:     event.VideoID,
				UserID:      event.UserID,
				OriginalKey: event.OriginalKey,
			}

			if err := c.usecase.Process(ctx, job); err != nil {
				log.Printf("falha ao processar vídeo %s: %v", event.VideoID, err)
				c.retryOrDeadLetter(msg, event)
				continue
			}

			_ = msg.Ack(false)
		}
	}
}

func (c *Consumer) retryOrDeadLetter(msg amqp091.Delivery, event videoUploadedEvent) {
	retryCount := getRetryCount(msg.Headers)

	if retryCount < maxRetries {
		if err := c.publishWithRetryHeader(retryQueueName, event, retryCount+1); err != nil {
			log.Printf("falha ao enfileirar retry: %v", err)
			_ = msg.Nack(false, true)
			return
		}
		_ = msg.Ack(false)
		return
	}

	if err := c.publishWithRetryHeader(dlqQueueName, event, retryCount); err != nil {
		log.Printf("falha ao publicar na DLQ: %v", err)
		_ = msg.Nack(false, true)
		return
	}
	_ = msg.Ack(false)
}

func (c *Consumer) publishWithRetryHeader(targetQueue string, event videoUploadedEvent, retryCount int) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return c.ch.Publish("", targetQueue, false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
		Headers:     amqp091.Table{retryHeader: int32(retryCount)},
	})
}

func getRetryCount(headers amqp091.Table) int {
	if headers == nil {
		return 0
	}
	switch v := headers[retryHeader].(type) {
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

func (c *Consumer) Close() {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
