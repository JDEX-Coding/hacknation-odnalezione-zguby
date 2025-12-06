package services

import (
	"encoding/json"
	"fmt"

	"github.com/hacknation/odnalezione-zguby/service-a-gateway/internal/storage"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

type RabbitMQConsumer struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	storage      *storage.PostgresStorage
	exchangeName string
	url          string
}

func NewRabbitMQConsumer(url, exchangeName string, storage *storage.PostgresStorage) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	consumer := &RabbitMQConsumer{
		conn:         conn,
		channel:      channel,
		storage:      storage,
		exchangeName: exchangeName,
		url:          url,
	}

	return consumer, nil
}

func (c *RabbitMQConsumer) Start() error {
	// Declare queue
	q, err := c.channel.QueueDeclare(
		"q.gateway.status_updates", // name
		true,                       // durable
		false,                      // delete when unused
		false,                      // exclusive
		false,                      // no-wait
		nil,                        // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind to item.embedded (Clip finished)
	err = c.channel.QueueBind(
		q.Name,
		"item.embedded",
		c.exchangeName,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue to item.embedded: %w", err)
	}

	// Bind to item.vectorized (Qdrant finished)
	err = c.channel.QueueBind(
		q.Name,
		"item.vectorized",
		c.exchangeName,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue to item.vectorized: %w", err)
	}

	msgs, err := c.channel.Consume(
		q.Name, // queue
		"",     // consumer tag
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	go c.consumeLoop(msgs)

	log.Info().Msg("Gateway RabbitMQ consumer started")
	return nil
}

func (c *RabbitMQConsumer) consumeLoop(msgs <-chan amqp.Delivery) {
	for d := range msgs {
		log.Info().Str("routing_key", d.RoutingKey).Msg("Received status update")

		// Both item.embedded and item.vectorized should have item_id (or ID)

		// Map generic JSON to find ID
		var data map[string]interface{}
		if err := json.Unmarshal(d.Body, &data); err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal update message")
			d.Nack(false, false)
			continue
		}

		// Extract ID
		var itemID string
		if id, ok := data["item_id"].(string); ok {
			itemID = id
		} else if id, ok := data["id"].(string); ok {
			itemID = id
		}

		// Also check nested "data" field (used by Qdrant service wrapper)
		if itemID == "" || (len(itemID) > 36 && itemID[0:4] == "vec_") {
			// If we got the wrapper ID (e.g. vec_UUID_...), check inside data map for the real itemID
			if nestedData, ok := data["data"].(map[string]interface{}); ok {
				if id, ok := nestedData["item_id"].(string); ok {
					itemID = id
				}
			}
		}

		if itemID == "" {
			log.Warn().Msg("Update message missing item_id")
			d.Ack(false)
			continue
		}

		// Update DB
		item, exists := c.storage.Get(itemID)
		if !exists {
			log.Warn().Str("id", itemID).Msg("Item not found for update")
			d.Ack(false)
			continue
		}

		changed := false
		if d.RoutingKey == "item.embedded" {
			if !item.ProcessedByClip {
				item.ProcessedByClip = true
				changed = true
			}
		} else if d.RoutingKey == "item.vectorized" {
			if !item.ProcessedByQdrant {
				item.ProcessedByQdrant = true
				if item.Status == "pending" {
					item.Status = "published"
				}
				changed = true
			}
		}

		if changed {
			if err := c.storage.Save(item); err != nil {
				log.Error().Err(err).Str("id", itemID).Msg("Failed to update item status")
				d.Nack(false, true) // retry
				continue
			}
			log.Info().Str("id", itemID).Msg("Updated item status")
		}

		d.Ack(false)
	}
}

func (c *RabbitMQConsumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
