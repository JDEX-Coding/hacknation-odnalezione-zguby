package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/hacknation/odnalezione-zguby/service-a-gateway/internal/handlers"
	"github.com/hacknation/odnalezione-zguby/service-a-gateway/internal/services"
	"github.com/hacknation/odnalezione-zguby/service-a-gateway/internal/storage"
)

func main() {
	// Setup logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("No .env file found, using environment variables")
	}

	config := loadConfig()

	log.Info().
		Str("host", config.Host).
		Str("port", config.Port).
		Msg("Starting Service A: Gateway")

	log.Info().Msg("Initializing MinIO storage...")
	minioStorage, err := storage.NewMinIOStorage(
		config.MinIOEndpoint,
		config.MinIOPublicEndpoint,
		config.MinIOAccessKey,
		config.MinIOSecretKey,
		config.MinIOBucket,
		config.MinIOUseSSL,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize MinIO storage")
	}
	log.Info().Msg("MinIO storage initialized successfully")

	log.Info().Msg("Initializing RabbitMQ publisher...")
	rabbitMQPublisher, err := services.NewRabbitMQPublisher(
		config.RabbitMQURL,
		config.RabbitMQExchange,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize RabbitMQ publisher")
	}
	defer rabbitMQPublisher.Close()
	log.Info().Msg("RabbitMQ publisher initialized successfully")

	log.Info().Msg("Initializing Vision API service...")
	visionService := services.NewVisionService(
		config.VisionAPIKey,
		config.VisionAPIEndpoint,
		config.VisionModel,
	)
	if config.VisionAPIKey == "" {
		log.Warn().Msg("Vision API key not configured - AI features will be disabled")
	}
	log.Info().Msg("Vision API service initialized")

	log.Info().Msg("Initializing Postgres storage...")
	itemStorage, err := storage.NewPostgresStorage(
		config.DBHost,
		config.DBPort,
		config.DBUser,
		config.DBPassword,
		config.DBName,
		config.DBSSLMode,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Postgres storage")
	}
	log.Info().Msg("Postgres storage initialized")

	log.Info().Msg("Initializing RabbitMQ consumer...")
	rabbitMQConsumer, err := services.NewRabbitMQConsumer(
		config.RabbitMQURL,
		config.RabbitMQExchange,
		itemStorage,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize RabbitMQ consumer")
	}
	defer rabbitMQConsumer.Close()

	if err := rabbitMQConsumer.Start(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start RabbitMQ consumer")
	}
	log.Info().Msg("RabbitMQ consumer initialized and started")

	log.Info().Msg("Initializing HTTP handlers...")
	handler, err := handlers.NewHandler(
		config.TemplatesPath,
		minioStorage,
		rabbitMQPublisher,
		visionService,
		itemStorage,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize handlers")
	}
	log.Info().Msg("HTTP handlers initialized successfully")

	// Setup router
	router := setupRouter(handler, config.StaticPath)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", config.Host, config.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().
			Str("address", srv.Addr).
			Msg("🚀 Server starting...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	log.Info().Msg("✅ Service A: Gateway is running")
	log.Info().Msgf("📍 Open http://localhost:%s in your browser", config.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited gracefully")
}

type Config struct {
	Host                string
	Port                string
	TemplatesPath       string
	StaticPath          string
	RabbitMQURL         string
	RabbitMQExchange    string
	MinIOEndpoint       string
	MinIOPublicEndpoint string
	MinIOAccessKey      string
	MinIOSecretKey      string
	MinIOBucket         string
	MinIOUseSSL         bool
	VisionAPIKey        string
	VisionAPIEndpoint   string
	VisionModel         string
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	DBSSLMode           string
}

// loadConfig loads configuration from environment variables
func loadConfig() *Config {
	return &Config{
		Host:                getEnv("GATEWAY_HOST", "0.0.0.0"),
		Port:                getEnv("GATEWAY_PORT", "8080"),
		TemplatesPath:       getEnv("TEMPLATES_PATH", "web/templates"),
		StaticPath:          getEnv("STATIC_PATH", "web/static"),
		RabbitMQURL:         getEnv("RABBITMQ_URL", "amqp://admin:admin123@localhost:5672/"),
		RabbitMQExchange:    getEnv("RABBITMQ_EXCHANGE", "lost-found.events"),
		MinIOEndpoint:       getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOPublicEndpoint: getEnv("MINIO_PUBLIC_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:      getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:      getEnv("MINIO_SECRET_KEY", "minioadmin123"),
		MinIOBucket:         getEnv("MINIO_BUCKET_NAME", "lost-items-images"),
		MinIOUseSSL:         getEnv("MINIO_USE_SSL", "false") == "true",
		VisionAPIKey:        getEnv("VISION_API_KEY", ""),
		VisionAPIEndpoint:   getEnv("VISION_API_ENDPOINT", "https://api.openai.com/v1"),
		VisionModel:         getEnv("VISION_MODEL", "gpt-4o"),
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "postgres"),
		DBPassword:          getEnv("DB_PASSWORD", "postgres"),
		DBName:              getEnv("DB_NAME", "postgres"),
		DBSSLMode:           getEnv("DB_SSL_MODE", "disable"),
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// setupRouter configures all routes and middleware
func setupRouter(h *handlers.Handler, staticPath string) *mux.Router {
	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(recoveryMiddleware)

	// Static files
	if _, err := os.Stat(staticPath); err == nil {
		fs := http.FileServer(http.Dir(staticPath))
		r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
		log.Info().Str("path", staticPath).Msg("Serving static files")
	}

	// Web routes
	r.HandleFunc("/", h.HomeHandler).Methods("GET")
	r.HandleFunc("/create", h.CreateFormHandler).Methods("GET")
	r.HandleFunc("/create", h.CreateHandler).Methods("POST")
	r.HandleFunc("/browse", h.BrowseHandler).Methods("GET")
	r.HandleFunc("/zguba/{id}", h.ItemDetailHandler).Methods("GET")
	r.HandleFunc("/search/semantic", h.SemanticSearchHandler).Methods("POST")

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/analyze-image", h.AnalyzeImageHandler).Methods("POST")
	api.HandleFunc("/analyze-image-form", h.AnalyzeImageFormHandler).Methods("POST")
	api.HandleFunc("/health", h.HealthCheckHandler).Methods("GET")

	// Health check at root
	r.HandleFunc("/health", h.HealthCheckHandler).Methods("GET")

	log.Info().Msg("Routes configured successfully")
	return r
}

// loggingMiddleware logs all HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wrapped.statusCode).
			Dur("duration_ms", time.Since(start)).
			Str("remote_addr", r.RemoteAddr).
			Msg("HTTP request")
	})
}

// recoveryMiddleware recovers from panics
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Interface("error", err).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Msg("Panic recovered")

				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
