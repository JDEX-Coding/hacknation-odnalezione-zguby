package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Configuration
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://admin:admin123@localhost:5672/")
	qdrantAddr := getEnv("QDRANT_ADDR", "localhost:6334")
	collectionName := getEnv("COLLECTION_NAME", "lost_items")

	// Initialize emulator
	emulator, err := NewEventEmulator(rabbitmqURL, qdrantAddr, collectionName)
	if err != nil {
		log.Fatalf("Failed to initialize emulator: %v", err)
	}
	defer emulator.Close()

	ctx := context.Background()

	// Show menu
	showMenu()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Main loop
	for {
		fmt.Print("\nSelect option (or 'q' to quit): ")
		var choice string
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			// Emit single event
			fmt.Println("\nSelect event type:")
			fmt.Println("  1. New Item")
			fmt.Println("  2. Vector Index")
			fmt.Println("  3. Search Result")
			fmt.Println("  4. Notification")
			fmt.Println("  5. Random")
			fmt.Print("Choice: ")
			var eventChoice string
			fmt.Scanln(&eventChoice)

			switch eventChoice {
			case "1":
				emulator.EmitNewItemEvent(ctx)
			case "2":
				emulator.EmitVectorIndexEvent(ctx)
			case "3":
				emulator.EmitSearchResultEvent(ctx)
			case "4":
				emulator.EmitNotificationEvent(ctx)
			case "5":
				emulator.EmitRandomEvent(ctx)
			default:
				fmt.Println("Invalid choice")
			}

		case "2":
			// Emit burst
			fmt.Print("Number of events: ")
			var count int
			fmt.Scanln(&count)
			fmt.Print("Delay between events (ms): ")
			var delay int
			fmt.Scanln(&delay)

			emulator.EmitBurstEvents(ctx, count, delay)

		case "3":
			// Continuous mode
			fmt.Print("Min delay (seconds): ")
			var minDelay int
			fmt.Scanln(&minDelay)
			fmt.Print("Max delay (seconds): ")
			var maxDelay int
			fmt.Scanln(&maxDelay)

			ctxWithCancel, cancel := context.WithCancel(ctx)
			go emulator.RunContinuous(ctxWithCancel,
				time.Duration(minDelay)*time.Second,
				time.Duration(maxDelay)*time.Second)

			fmt.Println("\nPress Enter to stop continuous mode...")
			fmt.Scanln()
			cancel()

		case "4":
			// End-to-end simulation
			emulator.SimulateEndToEndFlow(ctx)

		case "5":
			// Stress test
			fmt.Print("Duration (seconds): ")
			var duration int
			fmt.Scanln(&duration)
			fmt.Print("Events per second: ")
			var eventsPerSec int
			fmt.Scanln(&eventsPerSec)

			emulator.GenerateStressTest(ctx,
				time.Duration(duration)*time.Second,
				eventsPerSec)

		case "6":
			// Show queue stats
			emulator.ShowQueueStats()

		case "7":
			// Purge queues
			fmt.Print("Are you sure you want to purge all queues? (yes/no): ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm == "yes" {
				emulator.PurgeAllQueues()
			} else {
				fmt.Println("Cancelled")
			}

		case "8":
			// Custom message
			emitCustomMessagePrompt(ctx, emulator)

		case "q", "Q":
			fmt.Println("Goodbye!")
			return

		case "h", "H":
			showMenu()

		default:
			fmt.Println("Invalid option. Press 'h' for help.")
		}
	}
}

func showMenu() {
	fmt.Println("\n╔════════════════════════════════════════════════════╗")
	fmt.Println("║       RabbitMQ Event Emulator - Lost Items        ║")
	fmt.Println("╠════════════════════════════════════════════════════╣")
	fmt.Println("║ 1. Emit Single Event                               ║")
	fmt.Println("║ 2. Emit Burst of Events                            ║")
	fmt.Println("║ 3. Run Continuous Mode                             ║")
	fmt.Println("║ 4. Simulate End-to-End Flow                        ║")
	fmt.Println("║ 5. Stress Test                                     ║")
	fmt.Println("║ 6. Show Queue Statistics                           ║")
	fmt.Println("║ 7. Purge All Queues                                ║")
	fmt.Println("║ 8. Emit Custom Message                             ║")
	fmt.Println("║                                                    ║")
	fmt.Println("║ h. Show this menu                                  ║")
	fmt.Println("║ q. Quit                                            ║")
	fmt.Println("╚════════════════════════════════════════════════════╝")
}

func emitCustomMessagePrompt(ctx context.Context, emulator *EventEmulator) {
	fmt.Println("\nSelect queue:")
	fmt.Println("  1. Embedding Requests")
	fmt.Println("  2. Vector Indexing")
	fmt.Println("  3. Notifications")
	fmt.Println("  4. Search Results")
	fmt.Print("Choice: ")

	var queueChoice string
	fmt.Scanln(&queueChoice)

	var queueName QueueName
	switch queueChoice {
	case "1":
		queueName = QueueEmbeddingRequests
	case "2":
		queueName = QueueVectorIndexing
	case "3":
		queueName = QueueNotifications
	case "4":
		queueName = QueueSearchResults
	default:
		fmt.Println("Invalid choice")
		return
	}

	fmt.Print("Message type (e.g., new_item, update_item): ")
	var msgType string
	fmt.Scanln(&msgType)

	fmt.Print("Data (JSON key): ")
	var key string
	fmt.Scanln(&key)
	fmt.Print("Data (value): ")
	var value string
	fmt.Scanln(&value)

	data := map[string]interface{}{
		key: value,
	}

	emulator.EmitCustomMessage(ctx, queueName, MessageType(msgType), data)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
