// Package event publishes domain events to RabbitMQ following the
// envelope format described in docs/architecture/EVENT_DESIGN.md §3.
//
// Auth Service produces exactly one event in the MVP: UserRegistered
// (docs/architecture/EVENT_DESIGN.md §4, SERVICE_CATALOG.md §2), consumed
// by Notification Service to send a welcome email. Auth Service consumes
// no events.
package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Envelope is the common event wrapper shared across all CodeShop
// services (docs/architecture/EVENT_DESIGN.md §3).
type Envelope struct {
	EventID       string      `json:"event_id"`
	EventName     string      `json:"event_name"`
	EventVersion  int         `json:"event_version"`
	OccurredAt    time.Time   `json:"occurred_at"`
	CorrelationID string      `json:"correlation_id"`
	Producer      string      `json:"producer"`
	Payload       interface{} `json:"payload"`
}

// UserRegisteredPayload is the payload for the UserRegistered event.
type UserRegisteredPayload struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

// Publisher publishes events to the shared RabbitMQ exchange.
// docs/decisions/ADR-005-rabbitmq-messaging.md.
type Publisher interface {
	PublishUserRegistered(ctx context.Context, correlationID string, payload UserRegisteredPayload) error
	Close() error
}

type rabbitPublisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

// NewRabbitMQPublisher dials RabbitMQ and declares the shared topic
// exchange (idempotent: safe if another service already declared it).
func NewRabbitMQPublisher(url, exchange string) (Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	if err := ch.ExchangeDeclare(
		exchange, "topic", true, false, false, false, nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare exchange %q: %w", exchange, err)
	}

	return &rabbitPublisher{conn: conn, channel: ch, exchange: exchange}, nil
}

func (p *rabbitPublisher) PublishUserRegistered(ctx context.Context, correlationID string, payload UserRegisteredPayload) error {
	env := Envelope{
		EventID:       uuid.NewString(),
		EventName:     "UserRegistered",
		EventVersion:  1,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: correlationID,
		Producer:      "auth-service",
		Payload:       payload,
	}

	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal UserRegistered event: %w", err)
	}

	// Routing key convention: "<entity>.<event>" so consumers can bind
	// selectively (e.g. Notification Service binds "user.*", "order.*", ...).
	const routingKey = "user.registered"

	return p.channel.PublishWithContext(ctx, p.exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent, // survive broker restart, per at-least-once delivery (EVENT_DESIGN.md §7)
		Body:         body,
		MessageId:    env.EventID,
		Timestamp:    env.OccurredAt,
	})
}

func (p *rabbitPublisher) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
