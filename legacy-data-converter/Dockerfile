FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements
COPY requirements-basic.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements-basic.txt

# Copy application code
COPY text_extractor.py .
COPY nlp_converter.py .
COPY rabbitmq_handler.py .
COPY minio_handler.py .
COPY main.py .

# Environment variables
ENV PYTHONUNBUFFERED=1
ENV RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/
ENV RABBITMQ_EXCHANGE=lost-found.events
ENV MINIO_ENDPOINT=minio:9000
ENV MINIO_ACCESS_KEY=minioadmin
ENV MINIO_SECRET_KEY=minioadmin123
ENV MINIO_BUCKET_NAME=lost-items-images

# Run the application
CMD ["python", "main.py"]
