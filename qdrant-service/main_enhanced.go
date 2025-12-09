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
)

func main() {
	log.Println("=" + repeatString("=", 78))
	log.Println("🚀 Qdrant Service Enhanced - v2.0")
	log.Println("=" + repeatString("=", 78))
	log.Println("✨ Improvements:")
	log.Println("  - 512D embeddings (FIXED from 384D)")
	log.Println("  - Higher score threshold (0.75)")
	log.Println("  - Optimized HNSW configuration")
	log.Println("  - Hybrid search with filters")
	log.Println("  - Better search parameters")
	log.Println("=" + repeatString("=", 78))

	// Environment variables
	rabbitMQURL := getEnv("RABBITMQ_URL", "amqp://admin:admin123@rabbitmq:5672/")
	qdrantAddr := getEnv("QDRANT_ADDR", "qdrant-db:6334")
	collectionName := getEnv("COLLECTION_NAME", "lost_items_512")

	// Connect to RabbitMQ
	log.Println("Connecting to RabbitMQ...")
	rabbitMQ, err := NewRabbitMQHandler(rabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitMQ.Close()

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

	// Create collection with CORRECT 512 dimensions and optimized config
	ctx := context.Background()
	log.Println("Setting up Qdrant collection with ENHANCED configuration...")
	if err := qdrantHandler.CreateCollectionEnhanced(ctx); err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}

	log.Println("✅ Service initialized successfully!")
	log.Println("Starting to consume messages...")

	// Start consumers in separate goroutines
	go consumeVectorIndexing(rabbitMQ, qdrantHandler)
	go consumeSearchRequests(rabbitMQ, qdrantHandler)

	// Start HTTP server with enhanced endpoints
	go startHTTPServerEnhanced(qdrantHandler)

	// Show statistics periodically
	go showStats(rabbitMQ, qdrantHandler)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("🎧 Service is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("Shutting down gracefully...")
}

// SearchRequest represents enhanced search request
type SearchRequest struct {
	Embedding      []float32       `json:"embedding"`
	Limit          uint64          `json:"limit"`
	ScoreThreshold *float32        `json:"score_threshold,omitempty"`
	Filter         *EnhancedFilter `json:"filter,omitempty"`
}

// EnhancedFilter for hybrid search
type EnhancedFilter struct {
	Category string `json:"category,omitempty"`
	Location string `json:"location,omitempty"`
	DateFrom string `json:"date_from,omitempty"`
	DateTo   string `json:"date_to,omitempty"`
}

func startHTTPServerEnhanced(qdrant *QdrantHandler) {
	// Enhanced search endpoint
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

		// Validate embedding dimension
		if len(req.Embedding) != 512 {
			log.Printf("❌ Invalid embedding dimension: %d, expected 512", len(req.Embedding))
			http.Error(w, fmt.Sprintf("Invalid embedding dimension: %d, expected 512", len(req.Embedding)), http.StatusBadRequest)
			return
		}

		// Set defaults
		limit := req.Limit
		if limit == 0 {
			limit = 10
		}

		// ENHANCED: Higher default threshold for better quality
		scoreThreshold := float32(0.75)
		if req.ScoreThreshold != nil {
			scoreThreshold = *req.ScoreThreshold
			log.Printf("Using custom threshold: %.2f", scoreThreshold)
		}

		log.Printf("🔍 Search request: limit=%d, threshold=%.2f, dimension=%d", limit, scoreThreshold, len(req.Embedding))

		// Perform search with enhanced parameters
		results, err := qdrant.SearchSimilarEnhanced(
			context.Background(),
			req.Embedding,
			limit,
			scoreThreshold,
			req.Filter,
		)
		if err != nil {
			log.Printf("Search failed: %v", err)
			http.Error(w, "Search failed", http.StatusInternalServerError)
			return
		}

		log.Printf("✅ Found %d results", len(results))
		if len(results) > 0 {
			log.Printf("📊 Top score: %.4f", results[0].Score)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})

	// Health endpoint with diagnostics
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		info, err := qdrant.GetCollectionInfo(ctx)

		health := map[string]interface{}{
			"status":  "healthy",
			"version": "2.0",
			"features": map[string]bool{
				"512d_embeddings": true,
				"high_threshold":  true,
				"optimized_hnsw":  true,
				"hybrid_search":   true,
			},
		}

		if err == nil && info != nil {
			health["collection"] = map[string]interface{}{
				"name":          qdrant.collectionName,
				"vectors_count": info.VectorsCount,
				"points_count":  info.PointsCount,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	})

	log.Println("🌐 Starting enhanced HTTP server on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

func consumeVectorIndexing(rabbitMQ *RabbitMQHandler, qdrant *QdrantHandler) {
	msgs, err := rabbitMQ.ConsumeMessages(QueueLostItemsIngest, "ingest-consumer", false)
	if err != nil {
		log.Printf("Error starting ingest consumer: %v", err)
		return
	}

	log.Println("👂 Ingest consumer started - listening for item.embedded events")

	rabbitMQ.ProcessMessages(msgs, func(msg Message) error {
		log.Printf("📨 Processing item.embedded for message ID: %s", msg.ID)

		// Extract data from message
		itemID, ok := msg.Data["item_id"].(string)
		if !ok {
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

		log.Printf("Item: %s - %s", itemID, title)

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
			log.Printf("⚠️ No embedding found for item %s, skipping", itemID)
			return nil
		}

		// VALIDATE DIMENSION
		if len(embedding) != 512 {
			log.Printf("❌ Invalid embedding dimension: %d, expected 512", len(embedding))
			return fmt.Errorf("invalid embedding dimension: %d, expected 512", len(embedding))
		}

		log.Printf("✅ Embedding dimension validated: 512")

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

		log.Printf("✅ Indexed in Qdrant: %s (vector ID: %s)", itemID, vectorID)

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
			log.Printf("⚠️ Failed to publish item.vectorized event: %v", err)
		}

		return nil
	})
}

func consumeSearchRequests(rabbitMQ *RabbitMQHandler, qdrant *QdrantHandler) {
	msgs, err := rabbitMQ.ConsumeMessages(QueueEmbeddingRequests, "search-consumer", false)
	if err != nil {
		log.Printf("Error starting search consumer: %v", err)
		return
	}

	log.Println("👂 Search request consumer started")

	rabbitMQ.ProcessMessages(msgs, func(msg Message) error {
		log.Printf("🔍 Processing search request ID: %s", msg.ID)

		if text, ok := msg.Data["text"].(string); ok {
			log.Printf("Search query: %s", text)
		}

		return nil
	})
}

func showStats(rabbitMQ *RabbitMQHandler, qdrant *QdrantHandler) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("\n" + repeatString("=", 60))
		log.Println("📊 Service Statistics")
		log.Println(repeatString("=", 60))

		// Qdrant stats
		ctx := context.Background()
		if info, err := qdrant.GetCollectionInfo(ctx); err == nil && info != nil {
			log.Printf("🗄️  Collection: %s", qdrant.collectionName)
			log.Printf("    Points: %d", info.PointsCount)
			log.Printf("    Vectors: %d", info.VectorsCount)
		}

		// RabbitMQ stats
		queues := []QueueName{
			QueueLostItemsIngest,
			QueueLostItemsPublish,
			QueueEmbeddingRequests,
		}

		for _, queue := range queues {
			msgCount, consumerCount, err := rabbitMQ.GetQueueInfo(queue)
			if err != nil {
				continue
			}
			log.Printf("📬 %s: %d messages, %d consumers", queue, msgCount, consumerCount)
		}
		log.Println(repeatString("=", 60))
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
