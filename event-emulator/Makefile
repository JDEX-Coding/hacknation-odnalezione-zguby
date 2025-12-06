.PHONY: help install-air dev build clean test run

help: ## Show this help message
	@echo "Available targets:"
	@echo "  install-air    Install Air for live reloading"
	@echo "  dev            Run the emulator with Air (live reload)"
	@echo "  build          Build the emulator"
	@echo "  clean          Clean build artifacts"
	@echo "  run            Run the emulator directly"
	@echo "  deps           Download and tidy dependencies"

install-air: ## Install Air for live reloading
	go install github.com/cosmtrek/air@latest

dev: ## Run the emulator with Air (live reload)
	air

build: ## Build the emulator
	go build -o bin/emulator.exe ./emulator

clean: ## Clean build artifacts
	if exist tmp rmdir /s /q tmp
	if exist bin rmdir /s /q bin
	if exist build-errors.log del /q build-errors.log

test: ## Run tests
	go test -v ./...

deps: ## Download dependencies
	go mod download
	go mod tidy

run: ## Run the emulator directly
	go run ./emulator
