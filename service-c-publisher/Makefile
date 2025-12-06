.PHONY: build run clean test docker-build docker-run

# Binary name
BINARY_NAME=publisher

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v

run:
	$(GORUN) main.go

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test:
	$(GOTEST) -v ./...

deps:
	$(GOGET) -v -t -d ./...
	go mod tidy

docker-build:
	docker build -t odnalezione-publisher:latest .

docker-run:
	docker run --rm --env-file ../.env odnalezione-publisher:latest

# Development with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	air

.DEFAULT_GOAL := build
