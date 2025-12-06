package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// EventEmulator simulates various events for the lost items system
type EventEmulator struct {
	rabbitMQ *RabbitMQHandler
	qdrant   *QdrantHandler
}

// NewEventEmulator creates a new event emulator
func NewEventEmulator(rabbitmqURL, qdrantAddr, collectionName string) (*EventEmulator, error) {
	// Initialize RabbitMQ
	rabbitmq, err := NewRabbitMQHandler(rabbitmqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RabbitMQ: %w", err)
	}

	// Setup queues
	if err := rabbitmq.SetupQueues(); err != nil {
		rabbitmq.Close()
		return nil, fmt.Errorf("failed to setup queues: %w", err)
	}

	// Initialize Qdrant (optional, can be nil if not testing end-to-end)
	var qdrant *QdrantHandler
	if qdrantAddr != "" {
		qdrant, err = NewQdrantHandler(qdrantAddr, collectionName)
		if err != nil {
			log.Printf("Warning: Could not connect to Qdrant: %v", err)
		}
	}

	return &EventEmulator{
		rabbitMQ: rabbitmq,
		qdrant:   qdrant,
	}, nil
}

// Close closes all connections
func (e *EventEmulator) Close() error {
	if e.rabbitMQ != nil {
		e.rabbitMQ.Close()
	}
	if e.qdrant != nil {
		e.qdrant.Close()
	}
	return nil
}

// Sample data for generating realistic events
var (
	itemTitles = []string{
		"Lost Blue Backpack",
		"Lost Red Wallet",
		"Lost Black Phone",
		"Lost Keys with BMW Keychain",
		"Lost Laptop in Black Case",
		"Lost Sunglasses",
		"Lost Watch - Silver",
		"Lost Purple Umbrella",
		"Lost Green Jacket",
		"Lost White Headphones",
		"Lost Book - War and Peace",
		"Lost Credit Cards",
		"Lost Passport",
		"Lost Ring - Gold Band",
		"Lost Cat Toy",
	}

	itemDescriptions = []string{
		"Contains important documents and a laptop",
		"Brown leather with several credit cards inside",
		"iPhone 13 Pro with black case and screen protector",
		"Set of house and car keys with distinctive keychain",
		"Dell laptop with charger in protective case",
		"Ray-Ban sunglasses in original case",
		"Citizen watch with metal band, slight scratches",
		"Folding umbrella with floral pattern",
		"North Face jacket, size L, with zipper broken",
		"Sony wireless headphones with charging case",
		"Classic edition with worn cover",
		"Visa and Mastercard in name of John Smith",
		"Blue passport, expires 2027",
		"Wedding ring with engraving inside",
		"Small stuffed mouse, very worn",
	}

	categories = []string{
		"Bags",
		"Wallets",
		"Electronics",
		"Keys",
		"Clothing",
		"Accessories",
		"Documents",
		"Jewelry",
		"Books",
		"Toys",
	}

	locations = []string{
		"Central Train Station",
		"Shopping Mall - Food Court",
		"City Park near playground",
		"Coffee Shop on Main Street",
		"University Library 3rd Floor",
		"Bus Stop #42",
		"Airport Terminal B",
		"Metro Station - Red Line",
		"Gym Locker Room",
		"Restaurant - Pizza Palace",
		"Movie Theater - Screen 5",
		"Parking Garage Level 3",
		"Beach near lifeguard station",
		"Hotel Lobby",
		"Sports Stadium Gate C",
	}
)

// EmitNewItemEvent simulates a new lost item being reported
func (e *EventEmulator) EmitNewItemEvent(ctx context.Context) error {
	idx := rand.Intn(len(itemTitles))

	req := EmbeddingRequest{
		ItemID:      uuid.New().String(),
		Text:        itemTitles[idx],
		Description: itemDescriptions[idx],
		Category:    categories[rand.Intn(len(categories))],
	}

	log.Printf("📝 Emitting NEW ITEM event: %s", req.Text)
	return e.rabbitMQ.PublishEmbeddingRequest(ctx, req)
}

// EmitVectorIndexEvent simulates a vector being ready for indexing
func (e *EventEmulator) EmitVectorIndexEvent(ctx context.Context) error {
	idx := rand.Intn(len(itemTitles))

	// Generate random embedding (384 dimensions)
	embedding := make([]float32, 384)
	for i := range embedding {
		embedding[i] = rand.Float32()*2 - 1 // Random values between -1 and 1
	}

	req := VectorIndexRequest{
		ItemID:    uuid.New().String(),
		Embedding: embedding,
	}
	req.Payload.Title = itemTitles[idx]
	req.Payload.Description = itemDescriptions[idx]
	req.Payload.Category = categories[rand.Intn(len(categories))]
	req.Payload.Location = locations[rand.Intn(len(locations))]
	req.Payload.DateLost = time.Now().Add(-time.Duration(rand.Intn(720)) * time.Hour) // 0-30 days ago
	req.Payload.ContactInfo = fmt.Sprintf("user%d@example.com", rand.Intn(1000))

	log.Printf("🔢 Emitting VECTOR INDEX event: %s", req.Payload.Title)
	return e.rabbitMQ.PublishVectorIndexRequest(ctx, req)
}

// EmitSearchResultEvent simulates search results being found
func (e *EventEmulator) EmitSearchResultEvent(ctx context.Context) error {
	numResults := rand.Intn(5) + 1 // 1-5 results
	results := make([]interface{}, numResults)

	for i := 0; i < numResults; i++ {
		idx := rand.Intn(len(itemTitles))
		results[i] = map[string]interface{}{
			"item_id":     uuid.New().String(),
			"title":       itemTitles[idx],
			"description": itemDescriptions[idx],
			"category":    categories[rand.Intn(len(categories))],
			"location":    locations[rand.Intn(len(locations))],
			"score":       0.7 + rand.Float32()*0.3, // Score between 0.7 and 1.0
			"date_lost":   time.Now().Add(-time.Duration(rand.Intn(720)) * time.Hour).Format(time.RFC3339),
		}
	}

	searchResult := SearchResultMessage{
		QueryID:   uuid.New().String(),
		UserID:    fmt.Sprintf("user%d", rand.Intn(100)),
		Timestamp: time.Now(),
		Results:   results,
	}

	log.Printf("🔍 Emitting SEARCH RESULT event: %d results for user %s", numResults, searchResult.UserID)
	return e.rabbitMQ.PublishSearchResults(ctx, searchResult)
}

// EmitNotificationEvent simulates a notification being sent
func (e *EventEmulator) EmitNotificationEvent(ctx context.Context) error {
	notificationTypes := []string{"match_alert", "update", "reminder", "status_change"}
	titles := []string{
		"Potential Match Found!",
		"Item Status Updated",
		"Don't Forget to Check",
		"New Items Nearby",
	}
	messages := []string{
		"We found a potential match for your lost item. Check your dashboard!",
		"The status of your reported item has been updated.",
		"Remember to check if any new items match your search.",
		"New lost items were reported in your area.",
	}

	idx := rand.Intn(len(titles))

	notification := NotificationMessage{
		UserID:  fmt.Sprintf("user%d", rand.Intn(100)),
		Title:   titles[idx],
		Message: messages[idx],
		Type:    notificationTypes[rand.Intn(len(notificationTypes))],
		Time:    time.Now(),
	}

	log.Printf("🔔 Emitting NOTIFICATION event: %s for %s", notification.Title, notification.UserID)
	return e.rabbitMQ.PublishNotification(ctx, notification)
}

// EmitRandomEvent emits a random event type
func (e *EventEmulator) EmitRandomEvent(ctx context.Context) error {
	eventType := rand.Intn(4)

	switch eventType {
	case 0:
		return e.EmitNewItemEvent(ctx)
	case 1:
		return e.EmitVectorIndexEvent(ctx)
	case 2:
		return e.EmitSearchResultEvent(ctx)
	case 3:
		return e.EmitNotificationEvent(ctx)
	default:
		return e.EmitNewItemEvent(ctx)
	}
}

// EmitBurstEvents emits multiple events in quick succession
func (e *EventEmulator) EmitBurstEvents(ctx context.Context, count int, delayMs int) error {
	log.Printf("💥 Starting burst of %d events with %dms delay", count, delayMs)

	for i := 0; i < count; i++ {
		if err := e.EmitRandomEvent(ctx); err != nil {
			log.Printf("Error emitting event %d: %v", i, err)
		}

		if delayMs > 0 && i < count-1 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}

	log.Printf("✅ Burst complete!")
	return nil
}

// RunContinuous runs the emulator continuously with random intervals
func (e *EventEmulator) RunContinuous(ctx context.Context, minDelay, maxDelay time.Duration) {
	log.Printf("🚀 Starting continuous event emulation (delay: %v - %v)", minDelay, maxDelay)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Stopping event emulation")
			return
		case <-ticker.C:
			delay := minDelay + time.Duration(rand.Int63n(int64(maxDelay-minDelay)))
			time.Sleep(delay)

			if err := e.EmitRandomEvent(ctx); err != nil {
				log.Printf("Error emitting event: %v", err)
			}
		}
	}
}

// ShowQueueStats displays statistics for all queues
func (e *EventEmulator) ShowQueueStats() error {
	queues := []QueueName{
		QueueEmbeddingRequests,
		QueueVectorIndexing,
		QueueNotifications,
		QueueSearchResults,
	}

	fmt.Println("\n📊 Queue Statistics:")
	fmt.Println("═══════════════════════════════════════════════════")

	for _, queueName := range queues {
		msgCount, consumerCount, err := e.rabbitMQ.GetQueueInfo(queueName)
		if err != nil {
			log.Printf("Error getting stats for %s: %v", queueName, err)
			continue
		}

		fmt.Printf("%-25s | Messages: %4d | Consumers: %d\n",
			queueName, msgCount, consumerCount)
	}

	fmt.Println("═══════════════════════════════════════════════════")
	return nil
}

// SimulateEndToEndFlow simulates a complete workflow
func (e *EventEmulator) SimulateEndToEndFlow(ctx context.Context) error {
	log.Println("\n🎬 Starting End-to-End Flow Simulation")
	log.Println("───────────────────────────────────────")

	// Step 1: User reports lost item
	log.Println("Step 1: User reports a lost item")
	if err := e.EmitNewItemEvent(ctx); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	// Step 2: Embedding is generated and indexed
	log.Println("Step 2: System generates embedding and indexes")
	if err := e.EmitVectorIndexEvent(ctx); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	// Step 3: Another user searches
	log.Println("Step 3: Another user searches for their item")
	if err := e.EmitSearchResultEvent(ctx); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	// Step 4: Match notification sent
	log.Println("Step 4: Match notification sent to original user")
	if err := e.EmitNotificationEvent(ctx); err != nil {
		return err
	}

	log.Println("✅ End-to-End Flow Complete!")
	return nil
}

// GenerateStressTest generates high volume of events for testing
func (e *EventEmulator) GenerateStressTest(ctx context.Context, duration time.Duration, eventsPerSecond int) error {
	log.Printf("⚡ Starting stress test: %d events/sec for %v", eventsPerSecond, duration)

	interval := time.Second / time.Duration(eventsPerSecond)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeout := time.After(duration)
	eventCount := 0

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 Stress test cancelled. Sent %d events", eventCount)
			return ctx.Err()
		case <-timeout:
			log.Printf("✅ Stress test complete! Sent %d events in %v", eventCount, duration)
			return nil
		case <-ticker.C:
			if err := e.EmitRandomEvent(ctx); err != nil {
				log.Printf("Error in stress test: %v", err)
			}
			eventCount++
		}
	}
}

// EmitCustomMessage allows emitting custom messages for testing
func (e *EventEmulator) EmitCustomMessage(ctx context.Context, queueName QueueName, msgType MessageType, data map[string]interface{}) error {
	message := Message{
		ID:        uuid.New().String(),
		Type:      msgType,
		Timestamp: time.Now(),
		Data:      data,
		Priority:  5,
	}

	dataJSON, _ := json.MarshalIndent(data, "", "  ")
	log.Printf("📤 Emitting custom message to %s:\n%s", queueName, string(dataJSON))

	return e.rabbitMQ.PublishMessage(ctx, queueName, message)
}

// PurgeAllQueues clears all messages from all queues
func (e *EventEmulator) PurgeAllQueues() error {
	queues := []QueueName{
		QueueEmbeddingRequests,
		QueueVectorIndexing,
		QueueNotifications,
		QueueSearchResults,
	}

	log.Println("🗑️  Purging all queues...")
	totalPurged := 0

	for _, queueName := range queues {
		count, err := e.rabbitMQ.PurgeQueue(queueName)
		if err != nil {
			log.Printf("Error purging %s: %v", queueName, err)
			continue
		}
		totalPurged += count
		log.Printf("  Purged %d messages from %s", count, queueName)
	}

	log.Printf("✅ Total purged: %d messages\n", totalPurged)
	return nil
}
