# Test Publisher Service Locally

This guide helps you test the Publisher service without Docker.

## Prerequisites

1. RabbitMQ running (via Docker or local)
2. Go 1.21+ installed
3. Optional: Mock dane.gov.pl API server

## Step 1: Start Dependencies

```bash
# Start only RabbitMQ
docker compose up -d rabbitmq

# Wait for RabbitMQ to be ready
docker compose logs -f rabbitmq-init
```

## Step 2: Configure Environment

Create a `.env` file in the `service-c-publisher` directory:

```bash
RABBITMQ_URL=amqp://admin:admin123@localhost:5672/
RABBITMQ_EXCHANGE=lost-found.events
RABBITMQ_QUEUE=q.lost-items.publish
RABBITMQ_ROUTING_KEY=item.vectorized

# Point to mock server or real API
DANE_GOV_API_URL=http://localhost:8000
DANE_GOV_API_KEY=test-key

PUBLISHER_NAME=Test Municipality
PUBLISHER_ID=org-test-001
BASE_URL=http://localhost:8080
```

## Step 3: Run the Publisher

```bash
cd service-c-publisher
go run main.go
```

You should see:
```
Starting Service C: Publisher
Publisher service initialized successfully
Listening for messages on RabbitMQ...
```

## Step 4: Send Test Message

Use the event emulator or publish directly to RabbitMQ:

```bash
# Using rabbitmqadmin (if installed)
rabbitmqadmin publish exchange=lost-found.events routing_key=item.vectorized \
  payload='{"id":"test-123","title":"Test Item","description":"Test description","category":"Dokumenty","location":"Warsaw","found_date":"2024-12-01T10:00:00Z","reporting_date":"2024-12-01T10:00:00Z","image_url":"http://localhost:9000/test.jpg","contact_email":"test@example.com"}'
```

Or use Go code:

```go
package main

import (
    "encoding/json"
    "log"
    "time"
    amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
    conn, _ := amqp.Dial("amqp://admin:admin123@localhost:5672/")
    defer conn.Close()
    
    ch, _ := conn.Channel()
    defer ch.Close()
    
    event := map[string]interface{}{
        "id": "test-123",
        "title": "Test Lost Item",
        "description": "This is a test",
        "category": "Dokumenty",
        "location": "Warsaw",
        "found_date": time.Now(),
        "reporting_date": time.Now(),
        "image_url": "http://localhost:9000/test.jpg",
        "contact_email": "test@example.com",
        "timestamp": time.Now(),
    }
    
    body, _ := json.Marshal(event)
    
    ch.Publish("lost-found.events", "item.vectorized", false, false, 
        amqp.Publishing{
            ContentType: "application/json",
            Body: body,
        })
    
    log.Println("Message sent!")
}
```

## Step 5: Monitor Logs

The publisher will:
1. Receive the message
2. Format it to DCAT-AP
3. POST to dane.gov.pl API
4. Publish success event
5. ACK the message

## Mock dane.gov.pl API (Optional)

Create a simple mock server for testing:

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "github.com/google/uuid"
)

func main() {
    http.HandleFunc("/api/v1/datasets", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
            http.Error(w, "Method not allowed", 405)
            return
        }
        
        datasetID := uuid.New().String()
        response := map[string]interface{}{
            "data": map[string]interface{}{
                "id": datasetID,
                "type": "dataset",
                "attributes": map[string]interface{}{
                    "title": "Test Dataset",
                    "slug": "test-dataset",
                    "status": "published",
                    "url": "http://localhost:8000/datasets/" + datasetID,
                    "created": "2024-12-06T10:00:00Z",
                },
            },
            "links": map[string]string{
                "self": "http://localhost:8000/api/v1/datasets/" + datasetID,
            },
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
        log.Printf("Dataset created: %s", datasetID)
    })
    
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"status":"ok"}`))
    })
    
    log.Println("Mock dane.gov.pl API running on :8000")
    http.ListenAndServe(":8000", nil)
}
```

Save as `mock-api.go` and run:
```bash
go run mock-api.go
```

## Troubleshooting

**Publisher can't connect to RabbitMQ:**
```bash
# Check RabbitMQ is running
docker ps | grep rabbitmq

# Check logs
docker logs odnalezione-rabbitmq
```

**Messages not being consumed:**
```bash
# Check queue exists and has messages
curl -u admin:admin123 http://localhost:15672/api/queues/%2F/q.lost-items.publish
```

**API calls failing:**
```bash
# Test mock API
curl http://localhost:8000/health
```
