package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	qdrant "github.com/qdrant/go-client/qdrant"
)

func main() {
	log.Println("Starting Qdrant Service...")

	// Configuration from environment
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	qdrantAddr := getEnv("QDRANT_ADDR", "localhost:6334")
	collectionName := getEnv("COLLECTION_NAME", "lost_items")

	// Initialize RabbitMQ
	log.Println("Connecting to RabbitMQ...")
	rabbitMQ, err := NewRabbitMQHandler(rabbitmqURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitMQ.Close()

	// Setup queues
	log.Println("Setting up RabbitMQ queues...")
	if err := rabbitMQ.SetupQueues(); err != nil {
		log.Fatalf("Failed to setup queues: %v", err)
	}

	// Set QoS
	if err := rabbitMQ.SetQoS(10, 0, false); err != nil {
		log.Fatalf("Failed to set QoS: %v", err)
	}

	// Initialize Qdrant
	log.Println("Connecting to Qdrant...")
	qdrantHandler, err := NewQdrantHandler(qdrantAddr, collectionName)
	if err != nil {
		log.Fatalf("Failed to connect to Qdrant: %v", err)
	}
	defer qdrantHandler.Close()

	// Create collection if needed (384 dimensions for embeddings)
	ctx := context.Background()
	log.Println("Setting up Qdrant collection...")
	if err := qdrantHandler.CreateCollection(ctx, 384, qdrant.Distance_Cosine); err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}

	log.Println("Service initialized successfully!")
	log.Println("Starting to consume messages...")

	// Start consumers in separate goroutines
	go consumeVectorIndexing(rabbitMQ, qdrantHandler)
	go consumeSearchRequests(rabbitMQ, qdrantHandler)

	// Show statistics periodically
	go showStats(rabbitMQ, qdrantHandler)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Service is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("Shutting down gracefully...")
}

func consumeVectorIndexing(rabbitMQ *RabbitMQHandler, qdrant *QdrantHandler) {
	msgs, err := rabbitMQ.ConsumeMessages(QueueVectorIndexing, "indexing-consumer", false)
	if err != nil {
		log.Printf("Error starting vector indexing consumer: %v", err)
		return
	}

	log.Println("Vector indexing consumer started")

	rabbitMQ.ProcessMessages(msgs, func(msg Message) error {
		log.Printf("Processing vector indexing for message ID: %s", msg.ID)

		// Extract data from message
		itemID, ok := msg.Data["item_id"].(string)
		if !ok {
			return fmt.Errorf("invalid item_id")
		}

		// Extract embedding
		embeddingData, ok := msg.Data["embedding"].([]interface{})
		if !ok {
			return fmt.Errorf("invalid embedding data")
		}

		embedding := make([]float32, len(embeddingData))
		for i, v := range embeddingData {
			if fv, ok := v.(float64); ok {
				embedding[i] = float32(fv)
			}
		}

		// Extract payload
		payloadData, ok := msg.Data["payload"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid payload data")
		}

		payload := LostItemPayload{
			ItemID: itemID,
		}

		if title, ok := payloadData["title"].(string); ok {
			payload.Title = title
		}
		if desc, ok := payloadData["description"].(string); ok {
			payload.Description = desc
		}
		if loc, ok := payloadData["location"].(string); ok {
			payload.Location = loc
		}

		// Insert into Qdrant
		ctx := context.Background()
		_, err := qdrant.UpsertEmbedding(ctx, embedding, payload)
		if err != nil {
			return fmt.Errorf("failed to upsert to Qdrant: %w", err)
		}

		log.Printf("Indexed item: %s - %s", itemID, payload.Title)
		return nil
	})
}

func consumeSearchRequests(rabbitMQ *RabbitMQHandler, qdrant *QdrantHandler) {
	msgs, err := rabbitMQ.ConsumeMessages(QueueEmbeddingRequests, "search-consumer", false)
	if err != nil {
		log.Printf("Error starting search consumer: %v", err)
		return
	}

	log.Println("Search request consumer started")

	rabbitMQ.ProcessMessages(msgs, func(msg Message) error {
		log.Printf("Processing search/embedding request ID: %s", msg.ID)

		// In a real system, you would:
		// 1. Generate embedding from text using ML model
		// 2. Search Qdrant for similar vectors
		// 3. Publish results to search results queue

		// For now, just log the request
		if text, ok := msg.Data["text"].(string); ok {
			log.Printf("Request for: %s", text)
		}

		return nil
	})
}

func showStats(rabbitMQ *RabbitMQHandler, qdrant *QdrantHandler) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("\n=== Service Statistics ===")

		// RabbitMQ stats
		queues := []QueueName{
			QueueEmbeddingRequests,
			QueueVectorIndexing,
			QueueNotifications,
			QueueSearchResults,
		}

		for _, queue := range queues {
			msgCount, consumerCount, err := rabbitMQ.GetQueueInfo(queue)
			if err != nil {
				log.Printf("Error getting stats for %s: %v", queue, err)
				continue
			}
			log.Printf("  %s: %d messages, %d consumers", queue, msgCount, consumerCount)
		}

		log.Println("=============================\n")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
