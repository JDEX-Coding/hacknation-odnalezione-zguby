package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQHandler manages interactions with RabbitMQ message broker
type RabbitMQHandler struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// MessageType defines the type of message being sent
type MessageType string

const (
	MessageTypeNewItem      MessageType = "new_item"
	MessageTypeUpdateItem   MessageType = "update_item"
	MessageTypeDeleteItem   MessageType = "delete_item"
	MessageTypeSearchQuery  MessageType = "search_query"
	MessageTypeEmbedding    MessageType = "embedding_request"
	MessageTypeNotification MessageType = "notification"
)

// QueueName defines queue names for different purposes
type QueueName string

// Exchange and routing key constants
const (
	ExchangeLostFound    = "lost-found.events"
	RoutingKeySubmitted  = "item.submitted"
	RoutingKeyVectorized = "item.vectorized"
)

const (
	QueueLostItemsIngest   QueueName = "q.lost-items.ingest"
	QueueLostItemsPublish  QueueName = "q.lost-items.publish"
	QueueEmbeddingRequests QueueName = "embedding_requests"
	QueueVectorIndexing    QueueName = "vector_indexing"
	QueueNotifications     QueueName = "notifications"
	QueueSearchResults     QueueName = "search_results"
)

// Message represents a generic message structure
type Message struct {
	ID        string                 `json:"id"`
	Type      MessageType            `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Priority  uint8                  `json:"priority,omitempty"`
}

// EmbeddingRequest represents a request to generate embeddings
type EmbeddingRequest struct {
	ItemID      string `json:"item_id"`
	Text        string `json:"text"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// VectorIndexRequest represents a request to index a vector
type VectorIndexRequest struct {
	ItemID    string    `json:"item_id"`
	Embedding []float32 `json:"embedding"`
	Payload   struct {
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Category    string    `json:"category"`
		Location    string    `json:"location"`
		DateLost    time.Time `json:"date_lost"`
		ImageURL    string    `json:"image_url,omitempty"`
		ContactInfo string    `json:"contact_info,omitempty"`
	} `json:"payload"`
}

// SearchResultMessage represents search results to be sent
type SearchResultMessage struct {
	QueryID   string        `json:"query_id"`
	UserID    string        `json:"user_id"`
	Results   []interface{} `json:"results"`
	Timestamp time.Time     `json:"timestamp"`
}

// NotificationMessage represents a notification to be sent
type NotificationMessage struct {
	UserID  string    `json:"user_id"`
	Title   string    `json:"title"`
	Message string    `json:"message"`
	Type    string    `json:"type"`
	Time    time.Time `json:"time"`
}

// NewRabbitMQHandler creates a new RabbitMQ handler
func NewRabbitMQHandler(url string) (*RabbitMQHandler, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	handler := &RabbitMQHandler{
		conn:    conn,
		channel: channel,
	}

	return handler, nil
}

// Close closes the RabbitMQ connection and channel
func (h *RabbitMQHandler) Close() error {
	if h.channel != nil {
		if err := h.channel.Close(); err != nil {
			return err
		}
	}
	if h.conn != nil {
		if err := h.conn.Close(); err != nil {
			return err
		}
	}
	return nil
}

// SetupQueues declares all necessary queues and exchanges
func (h *RabbitMQHandler) SetupQueues() error {
	// Declare the topic exchange
	if err := h.DeclareExchange(ExchangeLostFound, "topic", true); err != nil {
		return err
	}

	// Declare queues for the new pattern
	if err := h.DeclareQueue(QueueLostItemsIngest, true, false); err != nil {
		return err
	}
	if err := h.BindQueue(QueueLostItemsIngest, ExchangeLostFound, RoutingKeySubmitted); err != nil {
		return err
	}

	if err := h.DeclareQueue(QueueLostItemsPublish, true, false); err != nil {
		return err
	}
	if err := h.BindQueue(QueueLostItemsPublish, ExchangeLostFound, RoutingKeyVectorized); err != nil {
		return err
	}

	// Keep backward compatibility with old queues
	queues := []QueueName{
		QueueEmbeddingRequests,
		QueueVectorIndexing,
		QueueNotifications,
		QueueSearchResults,
	}

	for _, queue := range queues {
		if err := h.DeclareQueue(queue, true, false); err != nil {
			return err
		}
	}

	return nil
}

// DeclareExchange declares an exchange
func (h *RabbitMQHandler) DeclareExchange(exchangeName, exchangeType string, durable bool) error {
	err := h.channel.ExchangeDeclare(
		exchangeName, // name
		exchangeType, // type (direct, fanout, topic, headers)
		durable,      // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange %s: %w", exchangeName, err)
	}
	log.Printf("Exchange '%s' declared successfully", exchangeName)
	return nil
}

// BindQueue binds a queue to an exchange with a routing key
func (h *RabbitMQHandler) BindQueue(queueName QueueName, exchangeName, routingKey string) error {
	err := h.channel.QueueBind(
		string(queueName), // queue name
		routingKey,        // routing key
		exchangeName,      // exchange
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue %s to exchange %s: %w", queueName, exchangeName, err)
	}
	log.Printf("Queue '%s' bound to exchange '%s' with routing key '%s'", queueName, exchangeName, routingKey)
	return nil
}

// DeclareQueue declares a queue with the given name and options
func (h *RabbitMQHandler) DeclareQueue(queueName QueueName, durable, autoDelete bool) error {
	_, err := h.channel.QueueDeclare(
		string(queueName), // name
		durable,           // durable
		autoDelete,        // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}
	log.Printf("Queue '%s' declared successfully", queueName)
	return nil
}

// PublishMessage publishes a message to the specified queue
func (h *RabbitMQHandler) PublishMessage(ctx context.Context, queueName QueueName, message Message) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = h.channel.PublishWithContext(
		ctx,
		"",                // exchange
		string(queueName), // routing key
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Priority:     message.Priority,
			Timestamp:    message.Timestamp,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published message %s to queue %s", message.ID, queueName)
	return nil
}

// PublishToExchange publishes a message to an exchange with a routing key
func (h *RabbitMQHandler) PublishToExchange(ctx context.Context, exchangeName, routingKey string, message Message) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = h.channel.PublishWithContext(
		ctx,
		exchangeName, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Priority:     message.Priority,
			Timestamp:    message.Timestamp,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish to exchange: %w", err)
	}

	log.Printf("Published message type '%s' to exchange '%s' with routing key '%s'", message.Type, exchangeName, routingKey)
	return nil
}

// GetQueueStats returns statistics about a queue
func (h *RabbitMQHandler) GetQueueStats(queueName QueueName) (messages, consumers int, err error) {
	queue, err := h.channel.QueueInspect(string(queueName))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to inspect queue %s: %w", queueName, err)
	}
	return queue.Messages, queue.Consumers, nil
}

// PurgeQueue purges all messages from a queue
func (h *RabbitMQHandler) PurgeQueue(queueName QueueName) (int, error) {
	count, err := h.channel.QueuePurge(string(queueName), false)
	if err != nil {
		return 0, fmt.Errorf("failed to purge queue %s: %w", queueName, err)
	}
	log.Printf("Purged queue '%s' (%d messages)", queueName, count)
	return count, nil
}

// PublishEmbeddingRequest publishes an embedding request to the queue
func (h *RabbitMQHandler) PublishEmbeddingRequest(ctx context.Context, req EmbeddingRequest) error {
	message := Message{
		ID:        req.ItemID,
		Type:      MessageTypeEmbedding,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"item_id":     req.ItemID,
			"text":        req.Text,
			"description": req.Description,
			"category":    req.Category,
		},
		Priority: 5,
	}

	return h.PublishMessage(ctx, QueueEmbeddingRequests, message)
}

// PublishItemSubmitted publishes an item.submitted event to the exchange
// This represents a new lost item submission from Service A (Gateway)
func (h *RabbitMQHandler) PublishItemSubmitted(ctx context.Context, req EmbeddingRequest) error {
	message := Message{
		ID:        req.ItemID,
		Type:      MessageTypeNewItem,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"item_id":     req.ItemID,
			"text":        req.Text,
			"description": req.Description,
			"category":    req.Category,
		},
		Priority: 5,
	}

	return h.PublishToExchange(ctx, ExchangeLostFound, RoutingKeySubmitted, message)
}

// PublishVectorIndexRequest publishes a vector indexing request to the queue
func (h *RabbitMQHandler) PublishVectorIndexRequest(ctx context.Context, req VectorIndexRequest) error {
	message := Message{
		ID:        req.ItemID,
		Type:      MessageTypeNewItem,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"item_id":   req.ItemID,
			"embedding": req.Embedding,
			"payload": map[string]interface{}{
				"title":        req.Payload.Title,
				"description":  req.Payload.Description,
				"category":     req.Payload.Category,
				"location":     req.Payload.Location,
				"date_lost":    req.Payload.DateLost,
				"image_url":    req.Payload.ImageURL,
				"contact_info": req.Payload.ContactInfo,
			},
		},
		Priority: 5,
	}

	return h.PublishMessage(ctx, QueueVectorIndexing, message)
}

// PublishSearchResults publishes search results to the queue
func (h *RabbitMQHandler) PublishSearchResults(ctx context.Context, result SearchResultMessage) error {
	message := Message{
		ID:        result.QueryID,
		Type:      MessageTypeSearchQuery,
		Timestamp: result.Timestamp,
		Data: map[string]interface{}{
			"query_id": result.QueryID,
			"user_id":  result.UserID,
			"results":  result.Results,
		},
		Priority: 7,
	}

	return h.PublishMessage(ctx, QueueSearchResults, message)
}

// PublishNotification publishes a notification to the queue
func (h *RabbitMQHandler) PublishNotification(ctx context.Context, notification NotificationMessage) error {
	message := Message{
		ID:        fmt.Sprintf("%s-%d", notification.UserID, notification.Time.Unix()),
		Type:      MessageTypeNotification,
		Timestamp: notification.Time,
		Data: map[string]interface{}{
			"user_id": notification.UserID,
			"title":   notification.Title,
			"message": notification.Message,
			"type":    notification.Type,
		},
		Priority: 8,
	}

	return h.PublishMessage(ctx, QueueNotifications, message)
}

// GetQueueInfo returns information about a queue (alias for GetQueueStats)
func (h *RabbitMQHandler) GetQueueInfo(queueName QueueName) (messages, consumers int, err error) {
	return h.GetQueueStats(queueName)
}
