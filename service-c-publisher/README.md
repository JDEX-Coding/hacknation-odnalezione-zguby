# Service C: Publisher

Publisher service that consumes vectorized lost items from RabbitMQ and publishes them to the dane.gov.pl API in DCAT-AP PL format.

## Features

- ✅ Consumes `item.vectorized` events from RabbitMQ
- ✅ Formats data to DCAT-AP PL standard
- ✅ Publishes datasets to dane.gov.pl API
- ✅ Publishes `item.published` success events
- ✅ Graceful shutdown handling
- ✅ Automatic retry on failure (via RabbitMQ nack/requeue)

## Architecture

```
RabbitMQ Queue              Publisher Service           dane.gov.pl API
q.lost-items.publish  -->   Consumer                --> POST /api/v1/datasets
(item.vectorized)      -->   ├─ DCAT Formatter      --> (JSON:API format)
                            └─ API Client          --> (With authentication)
                                 │
                                 ├─ Success --> Publish item.published
                                 └─ Error   --> Nack & Requeue
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RABBITMQ_URL` | `amqp://admin:admin123@localhost:5672/` | RabbitMQ connection URL |
| `RABBITMQ_EXCHANGE` | `lost-found.events` | Exchange name |
| `RABBITMQ_QUEUE` | `q.lost-items.publish` | Queue to consume from |
| `RABBITMQ_ROUTING_KEY` | `item.vectorized` | Routing key to bind |
| `DANE_GOV_API_URL` | `http://localhost:8000` | dane.gov.pl API base URL |
| `DANE_GOV_API_KEY` | `` | API authentication key |
| `PUBLISHER_NAME` | `Urząd Miasta - System Rzeczy Znalezionych` | Organization name |
| `PUBLISHER_ID` | `org-001` | Organization ID in dane.gov.pl |
| `BASE_URL` | `http://localhost:8080` | Base URL for dataset IDs |

## Running Locally

```bash
# Install dependencies
go mod download

# Run the service
make run

# Or with environment variables
DANE_GOV_API_URL=http://localhost:8000 go run main.go
```

## Running with Docker

```bash
# Build image
make docker-build

# Run container
docker run --rm \
  -e RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/ \
  -e DANE_GOV_API_URL=http://go-api-example:8000 \
  --network odnalezione-network \
  odnalezione-publisher:latest
```

## DCAT-AP PL Format

The service converts lost items to DCAT-AP PL standard:

```json
{
  "@context": "https://www.w3.org/ns/dcat",
  "@type": "dcat:Dataset",
  "@id": "http://localhost:8080/datasets/123",
  "dct:title": {"pl": "Znaleziony telefon"},
  "dct:description": {"pl": "Czarny iPhone znaleziony w parku"},
  "dct:publisher": {
    "@type": "foaf:Organization",
    "foaf:name": "Urząd Miasta"
  },
  "dcat:distribution": [{
    "@type": "dcat:Distribution",
    "dcat:accessURL": "http://minio:9000/lost-items-images/image.jpg"
  }]
}
```

## Message Flow

1. **Consume** `item.vectorized` event from queue
2. **Format** item data to DCAT-AP PL format
3. **Publish** to dane.gov.pl API endpoint
4. **Emit** `item.published` success event
5. **Acknowledge** message to RabbitMQ

On error:
- **Nack** message with requeue=true
- RabbitMQ will retry delivery

## Error Handling

- **Temporary failures**: Message is requeued for retry
- **Permanent failures**: Logged and acknowledged (DLQ should be configured)
- **API unavailable**: Automatic retry via RabbitMQ

## Development

```bash
# Run tests
make test

# Build binary
make build

# Clean build artifacts
make clean

# Update dependencies
make deps
```

## Integration

This service integrates with:
- **RabbitMQ**: Message queue (upstream from CLIP Worker)
- **dane.gov.pl API**: Government data portal (downstream)
- **Service A (Gateway)**: Shares event schema

## Monitoring

Logs include:
- Message consumption events
- API request/response details
- Success/failure rates
- Processing duration
