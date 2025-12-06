# Examples & Integration TestsService URLs:

🐇 RabbitMQ Management: http://localhost:15672 (admin/admin123)

Przykłady integracji poszczególnych serwisów z systemem "Odnalezione Zguby". 📦 MinIO Console: http://localhost:9001 (minioadmin/minioadmin123)

## 📍 Service URLs

| Serwis                  | URL                             | Credentials                |
| ----------------------- | ------------------------------- | -------------------------- |
| **RabbitMQ Management** | http://localhost:15672          | admin / admin123           |
| **MinIO Console**       | http://localhost:9001           | minioadmin / minioadmin123 |
| **Qdrant Dashboard**    | http://localhost:6333/dashboard | -                          |
| **Gateway UI**          | http://localhost:8080           | -                          |

## 🚀 Quick Start All Services

```bash
# Terminal 1: Start infrastructure
docker-compose up -d

# Terminal 2: Start Gateway
cd service-a-gateway
go run cmd/server/main.go

# Terminal 3: Start Event Emulator (for testing)
cd event-emulator
go run ./emulator
```

## 📚 Use Cases

### Use Case 1: Submit a Lost Item via Web UI

1. Open http://localhost:8080
2. Click "Zgłoś Przedmiot"
3. Upload a photo
4. Fill in the form
5. Submit
6. Watch RabbitMQ queue update
7. Verify in Qdrant dashboard

### Use Case 2: Test End-to-End Flow

```bash
# Start emulator
cd event-emulator
make run

# Select: 4. Simulate End-to-End Flow
# Watch flow:
# Gateway → RabbitMQ → CLIP Worker → Qdrant → Publisher → dane.gov.pl
```

### Use Case 3: Load Testing

```bash
# Start emulator
make run

# Select: 5. Stress Test
# Duration: 300 seconds
# Rate: 100 events/sec

# Monitor in another terminal
curl http://localhost:15672/api/queues -u admin:admin123 | jq
```

## 🔌 API Examples

### Python - Consume from RabbitMQ

```python
import pika
import json

connection = pika.BlockingConnection(
    pika.ConnectionParameters('localhost')
)
channel = connection.channel()

def callback(ch, method, properties, body):
    data = json.loads(body)
    print(f"Received item: {data['title']}")
    ch.basic_ack(delivery_tag=method.delivery_tag)

channel.basic_qos(prefetch_count=1)
channel.basic_consume(
    queue='q.lost-items.ingest',
    on_message_callback=callback
)

print('Waiting for messages...')
channel.start_consuming()
```

### Go - Publish to RabbitMQ

```go
package main

import (
    "encoding/json"
    "github.com/rabbitmq/amqp091-go"
)

func main() {
    conn, _ := amqp091.Dial("amqp://admin:admin123@localhost:5672/")
    ch, _ := conn.Channel()

    payload := map[string]interface{}{
        "id":           "550e8400-e29b-41d4-a716-446655440000",
        "title":        "Znaleziony portfel",
        "description":  "Czarny portfel skórzany",
        "category":     "Portfele i torby",
        "location":     "Warszawa",
        "image_url":    "http://minio:9000/lost-items-images/uploads/...",
        "contact_info": "biuro@urzad.pl",
    }

    body, _ := json.Marshal(payload)
    ch.Publish(
        "lost-found.events",
        "item.submitted",
        false,
        false,
        amqp091.Publishing{
            ContentType: "application/json",
            Body:        body,
        },
    )
}
```

### Query Qdrant Vectors

```bash
# List collections
curl http://localhost:6333/collections

# Search similar
curl -X POST http://localhost:6333/collections/lost_items/points/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3, ...],
    "limit": 10,
    "score_threshold": 0.7
  }'
```

## 📊 Monitoring & Debugging

### Check RabbitMQ Queues

```bash
# Via API
curl -u admin:admin123 http://localhost:15672/api/queues | jq

# Via CLI
docker exec -it odnalezione-rabbitmq rabbitmqctl list_queues name messages consumers
```

### Check Qdrant Collection

```bash
# Check collection info
curl http://localhost:6333/collections/lost_items | jq

# List points
curl -X POST http://localhost:6333/collections/lost_items/points/scroll \
  -H "Content-Type: application/json" \
  -d '{"limit": 10}'
```

### Check MinIO Uploads

```bash
# List uploads
docker exec -it odnalezione-minio mc ls myminio/lost-items-images

# Download file
docker exec -it odnalezione-minio mc cat myminio/lost-items-images/uploads/2025-12-06/uuid.jpg > file.jpg
```

## 🧪 Integration Tests

### Test 1: RabbitMQ Connectivity

```bash
docker exec -it odnalezione-rabbitmq rabbitmq-diagnostics -q ping
# Response: ok
```

### Test 2: Qdrant Connectivity

```bash
curl http://localhost:6333/health | jq
# Response: {"status": "ok"}
```

### Test 3: MinIO Connectivity

```bash
docker exec -it odnalezione-minio mc ls myminio
# Response: List of buckets
```

### Test 4: End-to-End Flow

```bash
# 1. Emit event
cd event-emulator && make run
# Select: 4. Simulate End-to-End Flow

# 2. Verify each step
# - Check queue in RabbitMQ UI
# - Check collection in Qdrant dashboard
# - Check bucket in MinIO UI
```

## 📝 Notes

-   All services communicate via RabbitMQ message broker
-   Embeddings are 384-dimensional CLIP vectors
-   MinIO stores all images for public access
-   dane.gov.pl integration pending (Service C - Publisher)
-   Event Emulator for testing without UI

---

Więcej szczegółów w głównym [README.md](../README.md)
