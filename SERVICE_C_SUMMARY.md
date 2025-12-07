# Service C: Publisher - Implementation Summary

## ✅ What Was Created

Service C (Publisher) is now fully implemented and ready to deploy. It completes the microservices architecture by consuming vectorized lost items from RabbitMQ and publishing them to the dane.gov.pl API.

## 📁 Project Structure

```
service-c-publisher/
├── main.go                              # Application entry point
├── Dockerfile                           # Docker image definition
├── Makefile                            # Build automation
├── go.mod                              # Go dependencies
├── README.md                           # Service documentation
├── TESTING.md                          # Local testing guide
├── mock-api.go                         # Mock dane.gov.pl API server
└── internal/
    ├── models/
    │   ├── events.go                   # Event definitions
    │   └── dcat.go                     # DCAT-AP data structures
    ├── consumer/
    │   └── rabbitmq_consumer.go        # RabbitMQ message consumption
    ├── formatter/
    │   └── dcat_formatter.go           # DCAT-AP PL formatter
    └── client/
        └── dane_gov_client.go          # dane.gov.pl API client
```

## 🎯 Key Features

### 1. **RabbitMQ Consumer** (`internal/consumer/`)
- ✅ Connects to RabbitMQ exchange
- ✅ Consumes `item.vectorized` events from `q.lost-items.publish` queue
- ✅ Manual acknowledgment (prevents message loss)
- ✅ QoS setting (processes one message at a time)
- ✅ Automatic requeue on failure
- ✅ Publishes `item.published` success events

### 2. **DCAT-AP PL Formatter** (`internal/formatter/`)
- ✅ Converts lost items to DCAT-AP standard
- ✅ Implements EU vocabulary mappings
- ✅ Multi-language support (Polish + English)
- ✅ Spatial/temporal metadata
- ✅ Theme and keyword extraction
- ✅ Distribution (image) handling
- ✅ JSON:API format for dane.gov.pl

### 3. **dane.gov.pl API Client** (`internal/client/`)
- ✅ HTTP client with timeouts
- ✅ Bearer token authentication
- ✅ Dataset publication endpoint
- ✅ Dataset retrieval endpoint
- ✅ Health check endpoint
- ✅ Error handling and logging

### 4. **Event Models** (`internal/models/`)
- ✅ `ItemVectorizedEvent` - Input from CLIP service
- ✅ `ItemPublishedEvent` - Success notification
- ✅ `DCATDataset` - Full DCAT-AP structure
- ✅ `DatasetRequest` - dane.gov.pl API format
- ✅ `DatasetResponse` - API response parsing

## 🔄 Message Flow

```
RabbitMQ Queue                Publisher Service              dane.gov.pl API
────────────────              ─────────────────              ───────────────
q.lost-items.publish    →     1. Consume Event         →     
(item.vectorized)       →     2. Parse JSON            →     
                              3. Format to DCAT-AP     →     
                              4. POST to API           →     POST /api/v1/datasets
                              5. Parse Response        ←     201 Created (Dataset)
                              6. Publish Success Event →     
                              7. ACK Message           →     
                              
RabbitMQ Exchange       ←     item.published event     ←     
```

## 🐳 Docker Integration

The service is added to `docker-compose.yml`:

```yaml
c-publisher:
    container_name: odnalezione-publisher
    build:
        context: ./service-c-publisher
    environment:
        - RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/
        - RABBITMQ_EXCHANGE=lost-found.events
        - RABBITMQ_QUEUE=q.lost-items.publish
        - RABBITMQ_ROUTING_KEY=item.vectorized
        - DANE_GOV_API_URL=${DANE_GOV_API_URL}
        - DANE_GOV_API_KEY=${DANE_GOV_API_KEY}
        - PUBLISHER_NAME=Urząd Miasta - System Rzeczy Znalezionych
        - PUBLISHER_ID=${PUBLISHER_ID}
    depends_on:
        - rabbitmq
    networks:
        - odnalezione-network
```

## 🚀 Running the Service

### **With Docker Compose**
```bash
# Start all services including publisher
docker compose up -d

# View publisher logs
docker compose logs -f c-publisher

# Restart publisher only
docker compose restart c-publisher
```

### **Locally (Development)**
```bash
cd service-c-publisher

# Set environment variables
export RABBITMQ_URL=amqp://admin:admin123@localhost:5672/
export DANE_GOV_API_URL=http://localhost:8000

# Run the service
go run main.go

# Or build and run
make build
./publisher
```

### **With Mock API (Testing)**
```bash
# Terminal 1: Start mock dane.gov.pl API
cd service-c-publisher
go run mock-api.go

# Terminal 2: Start publisher
go run main.go

# Terminal 3: Send test event
# (use event-emulator or RabbitMQ management UI)
```

## 📊 DCAT-AP PL Format Example

The service converts items like this:

**Input (ItemVectorizedEvent):**
```json
{
  "id": "abc-123",
  "title": "Znaleziony telefon",
  "description": "Czarny iPhone 13",
  "category": "Telefony",
  "location": "Park Łazienkowski, Warszawa",
  "found_date": "2024-12-01T14:00:00Z",
  "image_url": "http://minio:9000/lost-items-images/phone.jpg"
}
```

**Output (DCAT-AP PL):**
```json
{
  "@context": "https://www.w3.org/ns/dcat",
  "@type": "dcat:Dataset",
  "@id": "http://localhost:8080/datasets/abc-123",
  "dct:title": {
    "pl": "Znaleziony telefon",
    "en": "Lost item: Znaleziony telefon"
  },
  "dct:description": {
    "pl": "Czarny iPhone 13"
  },
  "dct:publisher": {
    "@type": "foaf:Organization",
    "foaf:name": "Urząd Miasta - System Rzeczy Znalezionych"
  },
  "dcat:theme": ["http://publications.europa.eu/resource/authority/data-theme/TECH"],
  "dcat:keyword": ["rzeczy znalezione", "lost and found", "Telefony"],
  "dct:spatial": {
    "@type": "dct:Location",
    "rdfs:label": "Park Łazienkowski, Warszawa"
  },
  "dcat:distribution": [{
    "@type": "dcat:Distribution",
    "dct:title": {"pl": "Zdjęcie rzeczy znalezionej"},
    "dcat:accessURL": "http://minio:9000/lost-items-images/phone.jpg",
    "dct:format": "image/jpeg"
  }]
}
```

## 🔧 Configuration

All configuration via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RABBITMQ_URL` | `amqp://admin:admin123@localhost:5672/` | RabbitMQ connection |
| `RABBITMQ_EXCHANGE` | `lost-found.events` | Exchange name |
| `RABBITMQ_QUEUE` | `q.lost-items.publish` | Queue to consume |
| `RABBITMQ_ROUTING_KEY` | `item.vectorized` | Routing key |
| `DANE_GOV_API_URL` | `http://localhost:8000` | API base URL |
| `DANE_GOV_API_KEY` | `` | API key (if required) |
| `PUBLISHER_NAME` | `Urząd Miasta...` | Organization name |
| `PUBLISHER_ID` | `org-001` | Organization ID |
| `BASE_URL` | `http://localhost:8080` | Dataset ID prefix |

## 🧪 Testing

See `TESTING.md` for detailed testing instructions.

**Quick test:**
```bash
# 1. Start dependencies
docker compose up -d rabbitmq

# 2. Start mock API
cd service-c-publisher
go run mock-api.go &

# 3. Start publisher
go run main.go &

# 4. Send test message via event-emulator or RabbitMQ UI
```

## 📝 Logging

The service provides structured logging:

```
2024-12-06T10:00:00Z INF 🚀 Starting Service C: Publisher
2024-12-06T10:00:01Z INF ✅ dane.gov.pl API is healthy
2024-12-06T10:00:01Z INF ✅ Publisher service initialized successfully
2024-12-06T10:00:01Z INF 🎧 Listening for messages on RabbitMQ...
2024-12-06T10:00:05Z INF 📨 Received message routing_key=item.vectorized message_id=abc-123
2024-12-06T10:00:05Z INF Processing item for publication item_id=abc-123 title="Znaleziony telefon"
2024-12-06T10:00:06Z INF Successfully published dataset to dane.gov.pl dataset_id=xyz-789 url=http://...
2024-12-06T10:00:06Z INF ✅ Successfully processed and published item item_id=abc-123 duration_ms=1234
```

## 🎉 Integration Complete

With Service C deployed, your complete architecture is now operational:

1. **Service A (Gateway)** → Accepts submissions, uploads images
2. **Service B (CLIP Worker)** → Generates embeddings, stores in Qdrant
3. **Service C (Publisher)** → Publishes to dane.gov.pl ✅ **NEW**
4. **Service D (Qdrant)** → Vector search capability

All connected via RabbitMQ event-driven architecture!

## 🔗 Related Files

- Main implementation: `service-c-publisher/main.go`
- Docker config: `docker-compose.yml`
- Environment template: `.env.example`
- Testing guide: `service-c-publisher/TESTING.md`
- Service README: `service-c-publisher/README.md`
