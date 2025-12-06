.PHONY: help build run stop clean logs install test setup-dev dev-up dev-down dev-logs test-setup

help:
	@echo "CLIP Service - Makefile Commands"
	@echo "================================="
	@echo ""
	@echo "MVP Development:"
	@echo "  make setup-dev  - Set up local development environment"
	@echo "  make dev-up     - Start infrastructure (RabbitMQ + MinIO)"
	@echo "  make dev-down   - Stop infrastructure"
	@echo "  make dev-logs   - View infrastructure logs"
	@echo "  make test-setup - Test local setup"
	@echo ""
	@echo "Production:"
	@echo "  make install    - Install Python dependencies (production)"
	@echo "  make build      - Build Docker image (production)"
	@echo "  make run        - Run service locally"
	@echo "  make docker-run - Run service in Docker"
	@echo "  make stop       - Stop Docker container"
	@echo "  make logs       - View Docker logs"
	@echo "  make clean      - Remove Docker image and volumes"
	@echo "  make test       - Run tests"

# MVP Development commands
setup-dev:
	@echo "Setting up local development environment..."
	python -m venv venv
	.\venv\Scripts\Activate.ps1 && pip install --upgrade pip && pip install -r requirements-dev.txt
	cp .env.local .env
	@echo "✅ Setup complete! Activate venv with: .\venv\Scripts\Activate.ps1"

dev-up:
	@echo "Starting development infrastructure..."
	docker-compose -f docker-compose.dev.yml up -d rabbitmq minio minio-init
	@echo "✅ Infrastructure started!"
	@echo "RabbitMQ Management: http://localhost:15672 (admin/admin123)"
	@echo "MinIO Console: http://localhost:9001 (minioadmin/minioadmin123)"

dev-down:
	docker-compose -f docker-compose.dev.yml down

dev-logs:
	docker-compose -f docker-compose.dev.yml logs -f

test-setup:
	@echo "Testing local setup..."
	python test_local_setup.py

# Production commands
install:
	pip install -r requirements.txt

build:
	docker build -t clip-service .

run:
	python main.py

docker-run:
	docker-compose up clip-service

stop:
	docker-compose stop clip-service

logs:
	docker-compose logs -f clip-service

clean:
	docker-compose down -v
	docker rmi clip-service || true

test:
	pytest tests/ -v
