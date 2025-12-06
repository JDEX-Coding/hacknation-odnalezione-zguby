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

const (
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

// PublishMessage publishes a message to a queue
func (h *RabbitMQHandler) PublishMessage(ctx context.Context, queueName QueueName, message Message) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = h.channel.PublishWithContext(
		ctx,
		"",                // exchange
		string(queueName), // routing key (queue name)
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // persistent messages
			Timestamp:    time.Now(),
			Priority:     message.Priority,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published message type '%s' to queue '%s'", message.Type, queueName)
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
			Timestamp:    time.Now(),
			Priority:     message.Priority,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish to exchange: %w", err)
	}

	log.Printf("Published message type '%s' to exchange '%s' with routing key '%s'", message.Type, exchangeName, routingKey)
	return nil
}

// ConsumeMessages starts consuming messages from a queue
func (h *RabbitMQHandler) ConsumeMessages(queueName QueueName, consumerTag string, autoAck bool) (<-chan amqp.Delivery, error) {
	msgs, err := h.channel.Consume(
		string(queueName), // queue
		consumerTag,       // consumer tag
		autoAck,           // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)

	if err != nil {
		return nil, fmt.Errorf("failed to start consuming: %w", err)
	}

	log.Printf("Started consuming from queue '%s' with consumer tag '%s'", queueName, consumerTag)
	return msgs, nil
}

// ProcessMessages processes messages from a channel with a handler function
func (h *RabbitMQHandler) ProcessMessages(msgs <-chan amqp.Delivery, handler func(Message) error) {
	for d := range msgs {
		var msg Message
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			d.Nack(false, false) // negative acknowledge, don't requeue
			continue
		}

		log.Printf("Received message: Type=%s, ID=%s", msg.Type, msg.ID)

		if err := handler(msg); err != nil {
			log.Printf("Error processing message: %v", err)
			d.Nack(false, true) // negative acknowledge, requeue
			continue
		}

		d.Ack(false) // acknowledge successful processing
	}
}

// PublishEmbeddingRequest publishes a request to generate embeddings
func (h *RabbitMQHandler) PublishEmbeddingRequest(ctx context.Context, req EmbeddingRequest) error {
	data := map[string]interface{}{
		"item_id":     req.ItemID,
		"text":        req.Text,
		"description": req.Description,
		"category":    req.Category,
	}

	message := Message{
		ID:        fmt.Sprintf("emb_%s_%d", req.ItemID, time.Now().Unix()),
		Type:      MessageTypeEmbedding,
		Timestamp: time.Now(),
		Data:      data,
		Priority:  5,
	}

	return h.PublishMessage(ctx, QueueEmbeddingRequests, message)
}

// PublishVectorIndexRequest publishes a request to index a vector
func (h *RabbitMQHandler) PublishVectorIndexRequest(ctx context.Context, req VectorIndexRequest) error {
	data := map[string]interface{}{
		"item_id":   req.ItemID,
		"embedding": req.Embedding,
		"payload":   req.Payload,
	}

	message := Message{
		ID:        fmt.Sprintf("idx_%s_%d", req.ItemID, time.Now().Unix()),
		Type:      MessageTypeNewItem,
		Timestamp: time.Now(),
		Data:      data,
		Priority:  7,
	}

	return h.PublishMessage(ctx, QueueVectorIndexing, message)
}

// PublishSearchResults publishes search results
func (h *RabbitMQHandler) PublishSearchResults(ctx context.Context, result SearchResultMessage) error {
	data := map[string]interface{}{
		"query_id":  result.QueryID,
		"user_id":   result.UserID,
		"results":   result.Results,
		"timestamp": result.Timestamp,
	}

	message := Message{
		ID:        fmt.Sprintf("search_%s_%d", result.QueryID, time.Now().Unix()),
		Type:      MessageTypeSearchQuery,
		Timestamp: time.Now(),
		Data:      data,
		Priority:  8,
	}

	return h.PublishMessage(ctx, QueueSearchResults, message)
}

// PublishNotification publishes a notification message
func (h *RabbitMQHandler) PublishNotification(ctx context.Context, notification NotificationMessage) error {
	data := map[string]interface{}{
		"user_id": notification.UserID,
		"title":   notification.Title,
		"message": notification.Message,
		"type":    notification.Type,
		"time":    notification.Time,
	}

	message := Message{
		ID:        fmt.Sprintf("notif_%s_%d", notification.UserID, time.Now().Unix()),
		Type:      MessageTypeNotification,
		Timestamp: time.Now(),
		Data:      data,
		Priority:  9,
	}

	return h.PublishMessage(ctx, QueueNotifications, message)
}

// SetupQueues declares all necessary queues for the application
func (h *RabbitMQHandler) SetupQueues() error {
	queues := []struct {
		name       QueueName
		durable    bool
		autoDelete bool
	}{
		{QueueEmbeddingRequests, true, false},
		{QueueVectorIndexing, true, false},
		{QueueNotifications, true, false},
		{QueueSearchResults, true, false},
	}

	for _, q := range queues {
		if err := h.DeclareQueue(q.name, q.durable, q.autoDelete); err != nil {
			return err
		}
	}

	return nil
}

// SetQoS sets the Quality of Service for the channel
func (h *RabbitMQHandler) SetQoS(prefetchCount, prefetchSize int, global bool) error {
	err := h.channel.Qos(
		prefetchCount, // prefetch count
		prefetchSize,  // prefetch size
		global,        // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}
	log.Printf("QoS set: prefetchCount=%d, prefetchSize=%d, global=%t", prefetchCount, prefetchSize, global)
	return nil
}

// GetQueueInfo retrieves information about a queue
func (h *RabbitMQHandler) GetQueueInfo(queueName QueueName) (int, int, error) {
	queue, err := h.channel.QueueInspect(string(queueName))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to inspect queue: %w", err)
	}
	return queue.Messages, queue.Consumers, nil
}

// PurgeQueue removes all messages from a queue
func (h *RabbitMQHandler) PurgeQueue(queueName QueueName) (int, error) {
	count, err := h.channel.QueuePurge(string(queueName), false)
	if err != nil {
		return 0, fmt.Errorf("failed to purge queue: %w", err)
	}
	log.Printf("Purged %d messages from queue '%s'", count, queueName)
	return count, nil
}
