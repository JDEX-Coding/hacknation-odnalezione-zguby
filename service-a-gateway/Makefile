.PHONY: help run build clean test deps install dev check health docker-build docker-run

# Default target
help:
	@echo "Service A: Gateway - Makefile Commands"
	@echo ""
	@echo "Available commands:"
	@echo "  make run          - Run the service"
	@echo "  make dev          - Run with hot reload (requires air)"
	@echo "  make build        - Build the binary"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make test         - Run tests"
	@echo "  make deps         - Download dependencies"
	@echo "  make install      - Install the binary"
	@echo "  make check        - Run linters and checks"
	@echo "  make health       - Check service health"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run in Docker"
	@echo ""

# Run the service
run:
	@echo "🚀 Starting Service A: Gateway..."
	@go run cmd/server/main.go

# Run with hot reload
dev:
	@echo "🔥 Starting with hot reload..."
	@air

# Build the binary
build:
	@echo "🔨 Building binary..."
	@go build -o bin/gateway cmd/server/main.go
	@echo "✅ Binary built: bin/gateway"


# Download dependencies
deps:
	@echo "📦 Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies ready"

# Install the binary
install: build
	@echo "📦 Installing binary..."
	@go install cmd/server/main.go
	@echo "✅ Binary installed"

# Docker run
docker-run:
	@echo "🐳 Running in Docker..."
	@docker run -p 8080:8080 --env-file .env service-a-gateway:latest

# Development cycle
watch:
	@echo "👀 Watching for changes..."
	@air

# Check dependencies for updates
deps-check:
	@echo "🔍 Checking for dependency updates..."
	@go list -u -m all

# Update dependencies
deps-update:
	@echo "⬆️  Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "✅ Dependencies updated"

# Show project structure
tree:
	@echo "📁 Project structure:"
	@tree -I 'bin|vendor|node_modules' -L 3 || ls -R
