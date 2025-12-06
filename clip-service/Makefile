.PHONY: help build run stop clean logs install test

help:
	@echo "CLIP Service - Makefile Commands"
	@echo "================================="
	@echo "make install    - Install Python dependencies"
	@echo "make build      - Build Docker image"
	@echo "make run        - Run service locally (requires RabbitMQ and Qdrant)"
	@echo "make docker-run - Run service in Docker"
	@echo "make stop       - Stop Docker container"
	@echo "make logs       - View Docker logs"
	@echo "make clean      - Remove Docker image and volumes"
	@echo "make test       - Run tests"

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
