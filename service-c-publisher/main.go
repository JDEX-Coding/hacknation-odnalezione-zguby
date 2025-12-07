package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/hacknation/odnalezione-zguby/service-c-publisher/internal/client"
	"github.com/hacknation/odnalezione-zguby/service-c-publisher/internal/consumer"
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
		config.DaneGovEmail,
		config.DaneGovPassword,
	)

	// Login to dane.gov.pl
	ctx := context.Background()
	log.Info().Msg("Logging in to dane.gov.pl...")
	if err := daneGovClient.Login(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to login to dane.gov.pl")
	}

	// Validate dataset ID is provided (only required for item->resource publishing)
	datasetID := config.DatasetID
	if datasetID == "" {
		log.Warn().Msg("DATASET_ID not set - item->resource publishing will be disabled, dataset publishing will still work")
	} else {
		log.Info().
			Str("dataset_id", datasetID).
			Msg("Using dataset for resource publishing")
	}

	// Initialize RabbitMQ consumer for items
	log.Info().Msg("Initializing RabbitMQ consumer for items...")
	itemConsumer, err := consumer.NewRabbitMQConsumer(
		config.RabbitMQURL,
		config.RabbitMQExchange,
		config.RabbitMQQueue,
		config.RabbitMQRoutingKey,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize RabbitMQ consumer for items")
	}
	defer itemConsumer.Close()

	// Initialize RabbitMQ consumer for datasets
	log.Info().Msg("Initializing RabbitMQ consumer for datasets...")
	datasetConsumer, err := consumer.NewRabbitMQConsumer(
		config.RabbitMQURL,
		config.RabbitMQExchange,
		"q.datasets.publish",
		"dataset.publish",
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize RabbitMQ consumer for datasets")
	}
	defer datasetConsumer.Close()

	log.Info().Msg("✅ Publisher service initialized successfully")

	// Create item message handler
	itemMessageHandler := func(event *models.ItemVectorizedEvent) error {
		log.Info().
			Str("item_id", event.ID).
			Str("title", event.Title).
			Str("category", event.Category).
			Msg("📝 Processing item for publication")

		// TODO: Add resource to dataset when API endpoint is available
		// Currently the mcod-api backend doesn't support resource creation via API
		// For now, we just log the item details
		log.Info().
			Str("item_id", event.ID).
			Str("title", event.Title).
			Str("category", event.Category).
			Str("location", event.Location).
			Str("image_url", event.ImageURL).
			Str("dataset_id", datasetID).
			Msg("✅ Item processed and logged (TODO: publish to dane.gov.pl when API is ready)")

		// Publish success event back to RabbitMQ
		publishedEvent := &models.ItemPublishedEvent{
			ID:              event.ID,
			DatasetID:       datasetID,
			PublishedAt:     time.Now(),
			DaneGovURL:      "", // TODO: Add URL when resource is actually created
			PublicationDate: time.Now(),
		}

		if err := itemConsumer.PublishPublishedEvent(ctx, publishedEvent); err != nil {
			log.Error().Err(err).Msg("Failed to publish success event")
			// Don't fail - item was already processed
		}

		return nil
	}

	// Create dataset message handler
	datasetMessageHandler := func(event *models.DatasetPublishEvent) error {
		log.Info().
			Str("dataset_id", event.DatasetID).
			Str("title", event.Title).
			Msg("📦 Processing dataset for publication to dane.gov.pl")

		// Create dataset request
		datasetReq := models.DatasetRequest{
			Data: models.DatasetData{
				Type: "dataset-submission",
				Attributes: models.DatasetAttributeDetail{
					Title:           event.Title,
					Notes:           event.Notes,
					URL:             event.URL,
					InstitutionName: event.InstitutionName,
					Email:           event.Email,
					Categories:      event.Categories,
					Tags:            event.Tags,
				},
			},
		}

		// Create dataset on dane.gov.pl
		response, err := daneGovClient.CreateDataset(ctx, datasetReq)
		if err != nil {
			log.Error().
				Err(err).
				Str("dataset_id", event.DatasetID).
				Msg("❌ Failed to create dataset on dane.gov.pl")
			return err
		}

		log.Info().
			Str("dataset_id", event.DatasetID).
			Str("dane_gov_id", response.Data.ID).
			Str("title", response.Data.Attributes.Title).
			Msg("✅ Dataset successfully created on dane.gov.pl")

		// Publish success event
		publishedEvent := &models.DatasetPublishedEvent{
			DatasetID:       event.DatasetID,
			DaneGovID:       response.Data.ID,
			PublishedAt:     time.Now(),
			DaneGovURL:      response.Links.Self,
			PublicationDate: time.Now(),
		}

		// Publish back to RabbitMQ
		publishedJSON, err := json.Marshal(publishedEvent)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal published event")
			return nil // Don't fail the consumer
		}

		if err := datasetConsumer.PublishEvent(ctx, publishedJSON, "dataset.published"); err != nil {
			log.Error().Err(err).Msg("Failed to publish dataset.published event")
			// Don't fail - dataset was already created
		}

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
		log.Info().Msg("🛑 Shutdown signal received, stopping consumers...")
		cancel()
	}()

	log.Info().Msg("🎧 Listening for messages on RabbitMQ...")

	// Start both consumers in goroutines
	go func() {
		log.Info().Msg("🎧 Item consumer started")
		if err := itemConsumer.Consume(consumerCtx, itemMessageHandler); err != nil {
			if err != context.Canceled {
				log.Error().Err(err).Msg("Item consumer error")
			}
		}
	}()

	go func() {
		log.Info().Msg("🎧 Dataset consumer started")
		if err := datasetConsumer.ConsumeDatasets(consumerCtx, datasetMessageHandler); err != nil {
			if err != context.Canceled {
				log.Error().Err(err).Msg("Dataset consumer error")
			}
		}
	}()

	// Wait for shutdown signal
	<-consumerCtx.Done()

	log.Info().Msg("👋 Publisher service stopped gracefully")
}

type Config struct {
	RabbitMQURL        string
	RabbitMQExchange   string
	RabbitMQQueue      string
	RabbitMQRoutingKey string
	DaneGovAPIURL      string
	DaneGovEmail       string
	DaneGovPassword    string
	PublisherName      string
	PublisherID        string
	BaseURL            string
	DatasetID          string
}

func loadConfig() *Config {
	return &Config{
		RabbitMQURL:        getEnv("RABBITMQ_URL", "amqp://admin:admin123@localhost:5672/"),
		RabbitMQExchange:   getEnv("RABBITMQ_EXCHANGE", "lost-found.events"),
		RabbitMQQueue:      getEnv("RABBITMQ_QUEUE", "q.lost-items.publish"),
		RabbitMQRoutingKey: getEnv("RABBITMQ_ROUTING_KEY", "item.vectorized"),
		DaneGovAPIURL:      getEnv("DANE_GOV_API_URL", "http://localhost:8000"),
		DaneGovEmail:       getEnv("DANE_GOV_EMAIL", "admin2@mcod.local"),
		DaneGovPassword:    getEnv("DANE_GOV_PASSWORD", "Hacknation-2025"),
		PublisherName:      getEnv("PUBLISHER_NAME", "Urząd Miasta - System Rzeczy Znalezionych"),
		PublisherID:        getEnv("PUBLISHER_ID", "org-001"),
		BaseURL:            getEnv("BASE_URL", "http://localhost:8080"),
		DatasetID:          getEnv("DATASET_ID", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
