package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/hacknation/odnalezione-zguby/service-c-publisher/internal/client"
	"github.com/hacknation/odnalezione-zguby/service-c-publisher/internal/consumer"
	"github.com/hacknation/odnalezione-zguby/service-c-publisher/internal/formatter"
	"github.com/hacknation/odnalezione-zguby/service-c-publisher/internal/models"
)

func main() {
	// Setup logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("No .env file found, using environment variables")
	}

	config := loadConfig()

	log.Info().Msg("🚀 Starting Service C: Publisher")
	log.Info().
		Str("dane_gov_url", config.DaneGovAPIURL).
		Str("publisher", config.PublisherName).
		Msg("Configuration loaded")

	// Initialize dane.gov.pl API client
	log.Info().Msg("Initializing dane.gov.pl API client...")
	daneGovClient := client.NewDaneGovClient(
		config.DaneGovAPIURL,
		config.DaneGovAPIKey,
	)

	// Check API health
	ctx := context.Background()
	if err := daneGovClient.HealthCheck(ctx); err != nil {
		log.Warn().Err(err).Msg("dane.gov.pl API health check failed (will retry during publish)")
	} else {
		log.Info().Msg("✅ dane.gov.pl API is healthy")
	}

	// Initialize DCAT formatter
	dcatFormatter := formatter.NewDCATFormatter(
		config.PublisherName,
		config.PublisherID,
		config.BaseURL,
	)

	// Initialize RabbitMQ consumer
	log.Info().Msg("Initializing RabbitMQ consumer...")
	rabbitConsumer, err := consumer.NewRabbitMQConsumer(
		config.RabbitMQURL,
		config.RabbitMQExchange,
		config.RabbitMQQueue,
		config.RabbitMQRoutingKey,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize RabbitMQ consumer")
	}
	defer rabbitConsumer.Close()

	log.Info().Msg("✅ Publisher service initialized successfully")

	// Create message handler
	messageHandler := func(event *models.ItemVectorizedEvent) error {
		log.Info().
			Str("item_id", event.ID).
			Str("title", event.Title).
			Msg("📝 Processing item for publication")

		// Format to dane.gov.pl format
		datasetRequest := dcatFormatter.FormatToDatasetRequest(event)

		// Publish to dane.gov.pl API
		response, err := daneGovClient.PublishDataset(ctx, datasetRequest)
		if err != nil {
			log.Error().
				Err(err).
				Str("item_id", event.ID).
				Msg("Failed to publish to dane.gov.pl")
			return err
		}

		// Publish success event back to RabbitMQ
		publishedEvent := &models.ItemPublishedEvent{
			ID:              event.ID,
			DatasetID:       response.Data.ID,
			PublishedAt:     time.Now(),
			DaneGovURL:      response.Data.Attributes.URL,
			PublicationDate: time.Now(),
		}

		if err := rabbitConsumer.PublishPublishedEvent(ctx, publishedEvent); err != nil {
			log.Error().Err(err).Msg("Failed to publish success event")
			// Don't fail - item was already published successfully
		}

		log.Info().
			Str("item_id", event.ID).
			Str("dataset_id", response.Data.ID).
			Str("url", response.Data.Attributes.URL).
			Msg("✅ Successfully published item to dane.gov.pl")

		return nil
	}

	// Start consuming messages
	consumerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info().Msg("🛑 Shutdown signal received, stopping consumer...")
		cancel()
	}()

	log.Info().Msg("🎧 Listening for messages on RabbitMQ...")

	if err := rabbitConsumer.Consume(consumerCtx, messageHandler); err != nil {
		if err != context.Canceled {
			log.Error().Err(err).Msg("Consumer error")
		}
	}

	log.Info().Msg("👋 Publisher service stopped gracefully")
}

type Config struct {
	RabbitMQURL        string
	RabbitMQExchange   string
	RabbitMQQueue      string
	RabbitMQRoutingKey string
	DaneGovAPIURL      string
	DaneGovAPIKey      string
	PublisherName      string
	PublisherID        string
	BaseURL            string
}

func loadConfig() *Config {
	return &Config{
		RabbitMQURL:        getEnv("RABBITMQ_URL", "amqp://admin:admin123@localhost:5672/"),
		RabbitMQExchange:   getEnv("RABBITMQ_EXCHANGE", "lost-found.events"),
		RabbitMQQueue:      getEnv("RABBITMQ_QUEUE", "q.lost-items.publish"),
		RabbitMQRoutingKey: getEnv("RABBITMQ_ROUTING_KEY", "item.vectorized"),
		DaneGovAPIURL:      getEnv("DANE_GOV_API_URL", "http://localhost:8000"),
		DaneGovAPIKey:      getEnv("DANE_GOV_API_KEY", ""),
		PublisherName:      getEnv("PUBLISHER_NAME", "Urząd Miasta - System Rzeczy Znalezionych"),
		PublisherID:        getEnv("PUBLISHER_ID", "org-001"),
		BaseURL:            getEnv("BASE_URL", "http://localhost:8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
