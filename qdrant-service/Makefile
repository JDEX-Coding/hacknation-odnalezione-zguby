.PHONY: help install-air dev build clean test run deps

help: ## Show this help message
	@echo "Available targets:"
	@echo "  install-air    Install Air for live reloading"
	@echo "  dev            Run the service with Air (live reload)"
	@echo "  build          Build the service"
	@echo "  clean          Clean build artifacts"
	@echo "  run            Run the service directly"
	@echo "  deps           Download and tidy dependencies"
	@echo "  example        Run the example usage"

install-air: ## Install Air for live reloading
	go install github.com/cosmtrek/air@latest

dev: ## Run the service with Air (live reload)
	air

build: ## Build the service
	go build -o bin/qdrant-service.exe .

clean: ## Clean build artifacts
	if exist tmp rmdir /s /q tmp
	if exist bin rmdir /s /q bin
	if exist build-errors.log del /q build-errors.log

test: ## Run tests
	go test -v ./...

deps: ## Download dependencies
	go mod download
	go mod tidy

run: ## Run the service directly
	go run .

example: ## Run the example usage
	go run example_usage.go qdrant-handler.go rabbitmq-handler.go
