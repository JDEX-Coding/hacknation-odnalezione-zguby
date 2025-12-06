package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

// RabbitMQPublisher handles publishing messages to RabbitMQ
type RabbitMQPublisher struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	exchangeName string
	url          string
}

// NewRabbitMQPublisher creates a new RabbitMQ publisher
func NewRabbitMQPublisher(url, exchangeName string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare the exchange (idempotent)
	err = channel.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	publisher := &RabbitMQPublisher{
		conn:         conn,
		channel:      channel,
		exchangeName: exchangeName,
		url:          url,
	}

	// Setup connection close handler
	go publisher.handleReconnect()

	log.Info().
		Str("exchange", exchangeName).
		Msg("RabbitMQ publisher initialized")

	return publisher, nil
}

// PublishItemSubmitted publishes an item.submitted event
func (p *RabbitMQPublisher) PublishItemSubmitted(ctx context.Context, event interface{}) error {
	return p.publish(ctx, "item.submitted", event)
}

// publish publishes a message to the exchange with the given routing key
func (p *RabbitMQPublisher) publish(ctx context.Context, routingKey string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(
		ctx,
		p.exchangeName, // exchange
		routingKey,     // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // persistent messages
			Body:         body,
			Timestamp:    time.Now(),
			MessageId:    fmt.Sprintf("%d", time.Now().UnixNano()),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Info().
		Str("routing_key", routingKey).
		Str("exchange", p.exchangeName).
		Int("body_size", len(body)).
		Msg("Message published to RabbitMQ")

	return nil
}

// handleReconnect handles automatic reconnection on connection loss
func (p *RabbitMQPublisher) handleReconnect() {
	closeChan := make(chan *amqp.Error)
	p.conn.NotifyClose(closeChan)

	for closeErr := range closeChan {
		if closeErr != nil {
			log.Error().
				Err(closeErr).
				Msg("RabbitMQ connection closed, attempting to reconnect...")

			// Attempt to reconnect with exponential backoff
			for {
				time.Sleep(5 * time.Second)

				conn, err := amqp.Dial(p.url)
				if err != nil {
					log.Error().Err(err).Msg("Failed to reconnect to RabbitMQ")
					continue
				}

				channel, err := conn.Channel()
				if err != nil {
					conn.Close()
					log.Error().Err(err).Msg("Failed to open channel")
					continue
				}

				// Redeclare exchange
				err = channel.ExchangeDeclare(
					p.exchangeName,
					"topic",
					true,
					false,
					false,
					false,
					nil,
				)
				if err != nil {
					channel.Close()
					conn.Close()
					log.Error().Err(err).Msg("Failed to declare exchange")
					continue
				}

				// Update connection and channel
				p.conn = conn
				p.channel = channel

				log.Info().Msg("Successfully reconnected to RabbitMQ")

				// Setup new close handler
				closeChan = make(chan *amqp.Error)
				p.conn.NotifyClose(closeChan)
				break
			}
		}
	}
}

// Close closes the RabbitMQ connection
func (p *RabbitMQPublisher) Close() error {
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close RabbitMQ channel")
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close RabbitMQ connection")
			return err
		}
	}
	log.Info().Msg("RabbitMQ publisher closed")
	return nil
}

// HealthCheck verifies the RabbitMQ connection
func (p *RabbitMQPublisher) HealthCheck() error {
	if p.conn == nil || p.conn.IsClosed() {
		return fmt.Errorf("RabbitMQ connection is closed")
	}
	if p.channel == nil {
		return fmt.Errorf("RabbitMQ channel is nil")
	}
	return nil
}
