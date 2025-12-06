# 🔄 Complete Event Flow Architecture

## Overview

This document describes the complete message flow through the system for processing lost items.

## 📊 System Architecture

```
┌─────────────────┐
│   User/Client   │
└────────┬────────┘
         │ HTTP POST /items
         ▼
┌─────────────────────────────────────────────────────────────┐
│                      A-GATEWAY SERVICE                      │
│  - Receives item submission                                 │
│  - Uploads image to MinIO                                   │
│  - Stores item metadata in PostgreSQL                       │
│  - Publishes event to RabbitMQ                             │
└────────┬────────────────────────────────────────────────────┘
         │
         │ Publishes: routing_key="item.submitted"
         ▼
┌─────────────────────────────────────────────────────────────┐
│                         RABBITMQ                            │
│  Exchange: lost-found.events (topic)                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Queue: q.lost-items.embed                          │   │
│  │  Binding: item.submitted                            │   │
│  └─────────────────────────────────────────────────────┘   │
└────────┬────────────────────────────────────────────────────┘
         │
         │ Consumes from q.lost-items.embed
         ▼
┌─────────────────────────────────────────────────────────────┐
│                     CLIP SERVICE                            │
│  - Consumes messages from q.lost-items.embed                │
│  - Downloads image from MinIO                               │
│  - Generates embeddings using CLIP model:                   │
│    • Text embedding (title + description + category)        │
│    • Image embedding (if image available)                   │
│  - Publishes enriched event to RabbitMQ                     │
└────────┬────────────────────────────────────────────────────┘
         │
         │ Publishes: routing_key="item.embedded"
         ▼
┌─────────────────────────────────────────────────────────────┐
│                         RABBITMQ                            │
│  Exchange: lost-found.events (topic)                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Queue: q.lost-items.ingest                         │   │
│  │  Binding: item.embedded                             │   │
│  └─────────────────────────────────────────────────────┘   │
└────────┬────────────────────────────────────────────────────┘
         │
         │ Consumes from q.lost-items.ingest
         ▼
┌─────────────────────────────────────────────────────────────┐
│                    QDRANT SERVICE                           │
│  - Consumes messages from q.lost-items.ingest               │
│  - Stores embeddings in Qdrant vector database              │
│  - Enables semantic search on stored items                  │
└────────┬────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│                      QDRANT DATABASE                        │
│  - Vector database for embeddings                           │
│  - Enables similarity search                                │
│  - Collection: lost_items                                   │
└─────────────────────────────────────────────────────────────┘
```

## 🔑 Message Flow Details

### Step 1: Item Submission (User → Gateway)

**HTTP Request:**

```http
POST /items
Content-Type: multipart/form-data

{
  "title": "Lost black backpack",
  "description": "Black Nike backpack with laptop",
  "category": "bags",
  "location": "Building A",
  "image": <binary file>
}
```

**Gateway Actions:**

1. Upload image to MinIO → get `image_key`
2. Store metadata in PostgreSQL
3. Publish event to RabbitMQ

### Step 2: Gateway → CLIP Service

**Published to:** Exchange `lost-found.events`
**Routing Key:** `item.submitted`
**Queue:** `q.lost-items.embed`

**Message Format:**

```json
{
    "id": "uuid-123",
    "title": "Lost black backpack",
    "description": "Black Nike backpack with laptop",
    "category": "bags",
    "location": "Building A",
    "image_key": "items/uuid-123.jpg",
    "created_at": "2025-12-06T10:00:00Z",
    "user_id": "user-456"
}
```

### Step 3: CLIP Service Processing

**CLIP Service Actions:**

1. **Consume** message from `q.lost-items.embed`
2. **Download** image from MinIO using `image_key`
3. **Generate embeddings:**
    - Text: Combine title + description + category → 512-dim vector
    - Image: Process image through CLIP → 512-dim vector
4. **Publish** enriched message with embeddings

### Step 4: CLIP Service → Qdrant Service

**Published to:** Exchange `lost-found.events`
**Routing Key:** `item.embedded`
**Queue:** `q.lost-items.ingest`

**Message Format:**

```json
{
  "id": "uuid-123",
  "title": "Lost black backpack",
  "description": "Black Nike backpack with laptop",
  "category": "bags",
  "location": "Building A",
  "image_key": "items/uuid-123.jpg",
  "created_at": "2025-12-06T10:00:00Z",
  "user_id": "user-456",
  "embedding": [0.123, -0.456, 0.789, ...], // 512 floats
  "metadata": {
    "has_image": true,
    "embedding_model": "openai/clip-vit-base-patch32",
    "processed_at": "2025-12-06T10:00:05Z"
  }
}
```

### Step 5: Qdrant Service Ingestion

**Qdrant Service Actions:**

1. **Consume** message from `q.lost-items.ingest`
2. **Validate** embedding vector
3. **Store** in Qdrant database:
    - Vector: The 512-dim embedding
    - Payload: All item metadata
4. **Index** for fast similarity search

## 📋 RabbitMQ Configuration

### Exchange

-   **Name:** `lost-found.events`
-   **Type:** `topic`
-   **Durable:** `true`

### Queues

| Queue Name            | Purpose                  | Bound To         | Consumer       |
| --------------------- | ------------------------ | ---------------- | -------------- |
| `q.lost-items.embed`  | Items awaiting embedding | `item.submitted` | CLIP Service   |
| `q.lost-items.ingest` | Items with embeddings    | `item.embedded`  | Qdrant Service |

### Routing Keys

| Key              | Publisher    | Description                    |
| ---------------- | ------------ | ------------------------------ |
| `item.submitted` | Gateway      | New item submitted by user     |
| `item.embedded`  | CLIP Service | Item with generated embeddings |

## 🔌 Service Connections

### Gateway Service

-   **Publishes to:** `lost-found.events` exchange
-   **Routing key:** `item.submitted`
-   **Dependency:** RabbitMQ, MinIO, PostgreSQL

### CLIP Service

-   **Consumes from:** `q.lost-items.embed`
-   **Publishes to:** `lost-found.events` exchange
-   **Routing key:** `item.embedded`
-   **Dependency:** RabbitMQ, MinIO

### Qdrant Service

-   **Consumes from:** `q.lost-items.ingest`
-   **Dependency:** RabbitMQ, Qdrant DB

## 🎯 Port Mapping

| Service        | Port(s)     | Purpose                   |
| -------------- | ----------- | ------------------------- |
| Gateway        | 8080        | HTTP API                  |
| CLIP Service   | -           | Internal only             |
| Qdrant Service | 8081        | Internal API (if exposed) |
| Qdrant DB      | 6333, 6334  | HTTP API, gRPC            |
| RabbitMQ       | 5672, 15672 | AMQP, Management UI       |
| MinIO          | 9000, 9001  | S3 API, Console           |
| PostgreSQL     | 5432        | Database                  |

## 🚀 Running the Complete System

### Start Everything

```bash
docker-compose up -d
```

### Verify Services

```bash
# Check all services are running
docker-compose ps

# Check RabbitMQ queues are created
# Open: http://localhost:15672 (admin/admin123)

# Check MinIO bucket exists
# Open: http://localhost:9001 (minioadmin/minioadmin123)
```

### Monitor Event Flow

```bash
# Watch CLIP service logs
docker-compose logs -f clip-service

# Watch Qdrant service logs
docker-compose logs -f qdrant-service

# Watch Gateway logs
docker-compose logs -f a-gateway
```

## 🧪 Testing the Complete Flow

### 1. Submit an Item

```bash
curl -X POST http://localhost:8080/items \
  -F "title=Lost black backpack" \
  -F "description=Black Nike backpack" \
  -F "category=bags" \
  -F "location=Building A" \
  -F "image=@test-image.jpg"
```

### 2. Check RabbitMQ

Open http://localhost:15672 and verify:

-   Message appears in `q.lost-items.embed`
-   Message is consumed by CLIP service
-   Message appears in `q.lost-items.ingest`
-   Message is consumed by Qdrant service

### 3. Check Qdrant

```bash
# Query Qdrant to verify item is indexed
curl http://localhost:6333/collections/lost_items/points
```

### 4. Check Logs

```bash
# CLIP Service should show:
# "Processing message for item: uuid-123"
# "Generated embeddings successfully"
# "Published to q.lost-items.ingest"

# Qdrant Service should show:
# "Received embedding for item: uuid-123"
# "Stored in Qdrant successfully"
```

## 🔍 Debugging

### Message Not Reaching CLIP

```bash
# Check if Gateway published
docker-compose logs a-gateway | grep "Published"

# Check RabbitMQ queue
# http://localhost:15672 → Queues → q.lost-items.embed
```

### CLIP Not Processing

```bash
# Check CLIP logs
docker-compose logs clip-service

# Check if CLIP can connect to RabbitMQ
docker-compose exec clip-service python -c "import pika; pika.BlockingConnection(pika.URLParameters('amqp://admin:admin123@rabbitmq:5672/'))"
```

### Embeddings Not Reaching Qdrant

```bash
# Check Qdrant logs
docker-compose logs qdrant-service

# Check RabbitMQ queue
# http://localhost:15672 → Queues → q.lost-items.ingest
```

## 📊 Expected Throughput

| Stage              | Time               | Cumulative   |
| ------------------ | ------------------ | ------------ |
| Gateway → RabbitMQ | < 100ms            | 100ms        |
| CLIP Processing    | 500-2000ms         | 600-2100ms   |
| CLIP → RabbitMQ    | < 100ms            | 700-2200ms   |
| Qdrant Ingestion   | < 200ms            | 900-2400ms   |
| **Total**          | **~1-2.5 seconds** | **per item** |

## 🎯 Key Benefits of This Architecture

1. **Decoupled Services** - Each service operates independently
2. **Asynchronous Processing** - Non-blocking item submission
3. **Fault Tolerance** - Messages persist in RabbitMQ if services are down
4. **Scalability** - Can add multiple CLIP/Qdrant service instances
5. **Observability** - Can monitor each stage via RabbitMQ UI
6. **Flexibility** - Easy to add more consumers/processors

## 🔄 Message Retry & Error Handling

-   **Dead Letter Queue:** Not configured yet (optional enhancement)
-   **Retry Logic:** CLIP service has built-in retry for MinIO downloads
-   **Message ACK:** Services only ACK after successful processing
-   **Persistence:** All queues are durable

## 📝 Configuration Files

-   **RabbitMQ Setup:** `rabbitmq-init.sh`
-   **Docker Compose:** `docker-compose.yml`
-   **CLIP Service:** `clip-service/main.py`
-   **Qdrant Service:** `qdrant-service/main.go`

---

**Summary:** Your system now has a complete event-driven architecture where Gateway → CLIP Service → Qdrant Service communicate asynchronously through RabbitMQ! 🎉
