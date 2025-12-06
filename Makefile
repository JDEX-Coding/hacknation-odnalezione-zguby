.PHONY: help up down restart logs logs-follow status clean init test ps health backup restore

tialize environment
init:
	@echo "🔧 Initializing environment..."
	@if
# Ini[ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ Created .env file from .env.example"; \
		echo "⚠️  Please edit .env with your API keys"; \
	else \
		echo "✅ .env file already exists"; \
	fi
	@echo "🚀 Starting services..."
	@docker-compose up -d
	@echo ""
	@echo "⏳ Waiting for services to be healthy..."
	@sleep 10
	@make status

# Start all services
up:
	@echo "🚀 Starting all services..."
	@docker-compose up -d
	@echo "✅ Services started"
	@echo ""
	@make status

# Stop all services
down:
	@echo "🛑 Stopping all services..."
	@docker-compose down
	@echo "✅ Services stopped"

# Restart all services
restart:
	@echo "🔄 Restarting all services..."
	@docker-compose restart
	@echo "✅ Services restarted"

# View logs
logs:
	@docker-compose logs --tail=100

# Follow logs in real-time
logs-follow:
	@docker-compose logs -f

# Show service status
status:
	@echo "📊 Service Status:"
	@echo ""
	@docker-compose ps
	@echo ""

# List running containers
ps:
	@docker-compose ps

# Check health of services
health:
	@echo "🏥 Health Check:"
	@echo ""
	@echo "RabbitMQ:"
	@docker inspect odnalezione-rabbitmq --format='  Status: {{.State.Health.Status}}' 2>/dev/null || echo "  ❌ Not running"
	@echo ""
	@echo "MinIO:"
	@docker inspect odnalezione-minio --format='  Status: {{.State.Health.Status}}' 2>/dev/null || echo "  ❌ Not running"
	@echo ""

# Clean everything (including volumes)
clean:
	@echo "⚠️  This will remove all containers and volumes (DATA WILL BE LOST)"
	@echo "Press Ctrl+C to cancel, or Enter to continue..."
	@read confirm
	@echo "🧹 Cleaning up..."
	@docker-compose down -v
	@echo "✅ Cleanup complete"

# Open RabbitMQ Management UI
rabbitmq-ui:
	@echo "🐇 Opening RabbitMQ Management UI..."
	@echo "URL: http://localhost:15672"
	@echo "Username: admin"
	@echo "Password: admin123"

# Open MinIO Console
minio-ui:
	@echo "📦 Opening MinIO Console..."
	@echo "URL: http://localhost:9001"
	@echo "Username: minioadmin"
	@echo "Password: minioadmin123"

# Individual service commands
rabbitmq-logs:
	@docker-compose logs -f rabbitmq

minio-logs:
	@docker-compose logs -f minio

rabbitmq-restart:
	@docker-compose restart rabbitmq

minio-restart:
	@docker-compose restart minio

# MinIO specific commands
minio-buckets:
	@echo "🪣 MinIO Buckets:"
	@docker exec odnalezione-minio mc ls myminio/

# Development helpers
dev-start: init
	@echo "🎯 Development environment ready!"
	@echo ""
	@echo "Services:"
	@echo "  RabbitMQ UI:  http://localhost:15672 (admin/admin123)"
	@echo "  MinIO UI:     http://localhost:9001 (minioadmin/minioadmin123)"
	@echo ""

dev-stop: down
	@echo "👋 Development environment stopped"
