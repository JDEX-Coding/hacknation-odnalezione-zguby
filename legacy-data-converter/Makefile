.PHONY: help build run test clean docker-build docker-run

help:
	@echo "Legacy Data Converter - Makefile commands:"
	@echo "  make build          - Install dependencies"
	@echo "  make build-full     - Install dependencies with NLP support"
	@echo "  make run            - Run the service locally"
	@echo "  make test           - Run tests"
	@echo "  make clean          - Clean temporary files"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run Docker container"

build:
	pip install -r requirements-basic.txt

build-full:
	pip install -r requirements.txt
	python -m spacy download pl_core_news_sm

run:
	python main.py

test:
	pytest -v

clean:
	rm -rf __pycache__
	rm -rf uploads/*
	rm -rf output/*
	rm -rf .pytest_cache
	find . -type f -name "*.pyc" -delete

docker-build:
	docker build -t legacy-data-converter .

docker-run:
	docker run -p 8083:8083 -v $(PWD)/uploads:/app/uploads -v $(PWD)/output:/app/output legacy-data-converter
