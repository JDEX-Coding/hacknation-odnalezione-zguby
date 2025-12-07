package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"

	"github.com/hacknation/odnalezione-zguby/service-c-publisher/internal/models"
)

// RabbitMQConsumer handles RabbitMQ message consumption
type RabbitMQConsumer struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	queueName    string
	exchangeName string
	routingKey   string
}

// NewRabbitMQConsumer creates a new RabbitMQ consumer
func NewRabbitMQConsumer(url, exchangeName, queueName, routingKey string) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS to process one message at a time
	if err := channel.Qos(1, 0, false); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	log.Info().
		Str("exchange", exchangeName).
		Str("queue", queueName).
		Str("routing_key", routingKey).
		Msg("RabbitMQ consumer initialized")

	return &RabbitMQConsumer{
		conn:         conn,
		channel:      channel,
		queueName:    queueName,
		exchangeName: exchangeName,
		routingKey:   routingKey,
	}, nil
}

// Consume starts consuming messages
func (c *RabbitMQConsumer) Consume(ctx context.Context, handler func(*models.ItemVectorizedEvent) error) error {
	msgs, err := c.channel.Consume(
		c.queueName,
		"",    // consumer tag
		false, // auto-ack (we'll ack manually)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Info().
		Str("queue", c.queueName).
		Msg("🎧 Started consuming messages from RabbitMQ")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Consumer context cancelled, stopping...")
			return ctx.Err()

		case msg, ok := <-msgs:
			if !ok {
				log.Warn().Msg("Message channel closed")
				return fmt.Errorf("message channel closed")
			}

			if err := c.processMessage(msg, handler); err != nil {
				log.Error().
					Err(err).
					Str("message_id", msg.MessageId).
					Msg("Failed to process message")

				// Reject and requeue if it's a temporary error
				if err := msg.Nack(false, true); err != nil {
					log.Error().Err(err).Msg("Failed to nack message")
				}
			} else {
				// Acknowledge successful processing
				if err := msg.Ack(false); err != nil {
					log.Error().Err(err).Msg("Failed to ack message")
				}
			}
		}
	}
}

// processMessage processes a single message
func (c *RabbitMQConsumer) processMessage(msg amqp.Delivery, handler func(*models.ItemVectorizedEvent) error) error {
	log.Info().
		Str("routing_key", msg.RoutingKey).
		Str("message_id", msg.MessageId).
		Int("body_size", len(msg.Body)).
		Msg("📨 Received message")

	var event models.ItemVectorizedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	log.Info().
		Str("item_id", event.ID).
		Str("title", event.Title).
		Str("category", event.Category).
		Msg("Processing item for publication")

	startTime := time.Now()
	if err := handler(&event); err != nil {
		return fmt.Errorf("handler error: %w", err)
	}

	log.Info().
		Str("item_id", event.ID).
		Dur("duration_ms", time.Since(startTime)).
		Msg("✅ Successfully processed and published item")

	return nil
}

// PublishPublishedEvent publishes a success event back to RabbitMQ
func (c *RabbitMQConsumer) PublishPublishedEvent(ctx context.Context, event *models.ItemPublishedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = c.channel.PublishWithContext(
		ctx,
		c.exchangeName,
		"item.published", // routing key for published events
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			MessageId:    event.ID,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Info().
		Str("item_id", event.ID).
		Str("dataset_id", event.DatasetID).
		Msg("📤 Published item.published event")

	return nil
}

// Close closes the consumer connection
func (c *RabbitMQConsumer) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close channel")
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close connection")
			return err
		}
	}
	log.Info().Msg("RabbitMQ consumer closed")
	return nil
}
