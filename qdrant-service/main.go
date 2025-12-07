package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	// Create collection if needed (512 dimensions for CLIP ViT-B/32)
	ctx := context.Background()
	log.Println("Setting up Qdrant collection...")
	if err := qdrantHandler.CreateCollection(ctx, 512, qdrant.Distance_Cosine); err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}

	log.Println("Service initialized successfully!")
	log.Println("Starting to consume messages...")

	// Start consumers in separate goroutines
	go consumeVectorIndexing(rabbitMQ, qdrantHandler)
	go consumeSearchRequests(rabbitMQ, qdrantHandler)

	// Start HTTP server
	go startHTTPServer(qdrantHandler)

	// Show statistics periodically
	go showStats(rabbitMQ, qdrantHandler)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Service is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("Shutting down gracefully...")
}

type SearchRequest struct {
	Embedding []float32 `json:"embedding"`
	Limit     uint64    `json:"limit"`
}

func startHTTPServer(qdrant *QdrantHandler) {
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if len(req.Embedding) == 0 {
			http.Error(w, "Embedding is required", http.StatusBadRequest)
			return
		}

		limit := req.Limit
		if limit == 0 {
			limit = 10
		}

		results, err := qdrant.SearchSimilar(context.Background(), req.Embedding, limit, 0.6)
		if err != nil {
			log.Printf("Search failed: %v", err)
			http.Error(w, "Search failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})

	log.Println("Starting HTTP server on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

func consumeVectorIndexing(rabbitMQ *RabbitMQHandler, qdrant *QdrantHandler) {
	// ... (content matches previous state)
	// Consume from q.lost-items.ingest (receives item.embedded events)
	msgs, err := rabbitMQ.ConsumeMessages(QueueLostItemsIngest, "ingest-consumer", false)
	if err != nil {
		log.Printf("Error starting ingest consumer: %v", err)
		return
	}

	log.Println("Ingest consumer started - listening for item.embedded events")

	rabbitMQ.ProcessMessages(msgs, func(msg Message) error {
		log.Printf("Processing item.embedded for message ID: %s", msg.ID)

		// Extract data from message
		itemID, ok := msg.Data["item_id"].(string)
		if !ok {
			// Try "id" as fallback
			itemID, ok = msg.Data["id"].(string)
			if !ok {
				return fmt.Errorf("invalid or missing 'item_id' field")
			}
		}

		title, _ := msg.Data["title"].(string)
		description, _ := msg.Data["description"].(string)
		category, _ := msg.Data["category"].(string)
		location, _ := msg.Data["location"].(string)
		imageURL, _ := msg.Data["image_url"].(string)
		contactInfo, _ := msg.Data["contact_info"].(string)

		log.Printf("Processing item: %s - %s", itemID, title)

		// Extract embedding
		var embedding []float32
		if embInterface, ok := msg.Data["embedding"].([]interface{}); ok {
			embedding = make([]float32, len(embInterface))
			for i, v := range embInterface {
				if f, ok := v.(float64); ok {
					embedding[i] = float32(f)
				}
			}
		}

		if len(embedding) == 0 {
			log.Printf("Warning: No embedding found for item %s, skipping indexing", itemID)
			return nil
		}

		// Create payload for Qdrant
		payload := LostItemPayload{
			ItemID:      itemID,
			Title:       title,
			Description: description,
			Category:    category,
			Location:    location,
			ImageURL:    imageURL,
			ContactInfo: contactInfo,
		}

		// Insert into Qdrant
		ctx := context.Background()
		vectorID, err := qdrant.UpsertEmbedding(ctx, embedding, payload)
		if err != nil {
			return fmt.Errorf("failed to upsert to Qdrant: %w", err)
		}

		log.Printf("Successfully indexed item in Qdrant: %s (vector ID: %s)", itemID, vectorID)

		// Publish item.vectorized event
		payloadMap := map[string]interface{}{
			"title":        title,
			"description":  description,
			"category":     category,
			"location":     location,
			"image_url":    imageURL,
			"contact_info": contactInfo,
		}

		if err := rabbitMQ.PublishItemVectorized(ctx, itemID, vectorID, payloadMap); err != nil {
			log.Printf("Warning: Failed to publish item.vectorized event: %v", err)
			// Don't return error, as the main indexing succeeded
		}

		log.Printf("Published item.vectorized event for: %s", itemID)
		return nil
	})
}

// generateMockEmbedding generates a mock embedding vector
// In production, this would call an actual embedding model
func generateMockEmbedding(dimensions int) []float32 {
	embedding := make([]float32, dimensions)
	for i := range embedding {
		// Simple mock: use index as value for demonstration
		embedding[i] = float32(i) / float32(dimensions)
	}
	return embedding
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
			QueueLostItemsIngest,
			QueueLostItemsPublish,
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
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
