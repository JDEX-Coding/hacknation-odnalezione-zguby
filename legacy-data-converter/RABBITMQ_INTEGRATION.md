# Legacy Data Converter - RabbitMQ Integration

## Overview

The legacy-data-converter service now operates as a **RabbitMQ consumer** instead of providing a REST API. It processes files from the `q.datasets.process` queue and publishes each converted item to the processing pipeline.

## Architecture Changes

### Previous (REST API):
```
User → HTTP POST /convert → Service → Response
```

### Current (RabbitMQ Consumer):
```
Gateway → [dataset.submitted] → q.datasets.process → Legacy Converter
Legacy Converter → [item.submitted] → q.lost-items.embed → CLIP Service
CLIP Service → [item.embedded] → q.lost-items.ingest → Qdrant Service
Qdrant Service → [item.vectorized] → q.lost-items.publish → Publisher Service
```

## Message Flow

### 1. Input Message (q.datasets.process)

The service consumes messages from `q.datasets.process` with the following format:

```json
{
  "dataset_id": "uuid-v4",
  "file_data": "base64-encoded-file-content",
  "file_name": "lost_items.csv",
  "file_format": ".csv"
}
```

**Fields:**
- `dataset_id` - UUID of the dataset (optional, used to associate items)
- `file_data` - Base64-encoded file content
- `file_name` - Original filename
- `file_format` - File extension (.csv, .json, .xml, .pdf, etc.)

**Routing Key:** `dataset.uploaded`

### 2. Processing

The service:
1. Decodes the base64 file data
2. Saves to temporary file
3. Extracts text using `text_extractor.py`
4. Converts to lost-items schema using `nlp_converter.py`
5. Handles multiple items (e.g., CSV with multiple rows)
6. Publishes each item individually

### 3. Output Messages (item.submitted)

For each converted item, publishes to exchange with routing key `item.submitted`:

```json
{
  "item_id": "uuid-v4",
  "text": "Portfel męski",
  "description": "Czarny portfel skórzany z dokumentami",
  "category": "portfele",
  "location": "ul. Marszałkowska 10, Warszawa",
  "date_lost": "2024-01-15T00:00:00",
  "reporting_date": "2024-01-16T00:00:00",
  "reporting_location": "Urząd Miasta",
  "image_url": "",
  "image_key": "",
  "contact_email": "biuro@urzad.pl",
  "contact_phone": "22-123-45-67",
  "timestamp": "2024-01-16T10:30:00"
}
```

**Routing Key:** `item.submitted`
**Queue:** `q.lost-items.embed` (consumed by CLIP service)

## RabbitMQ Configuration

### Exchange
- **Name:** `lost-found.events`
- **Type:** `topic`
- **Durable:** `true`

### Queues

| Queue Name | Consumer | Routing Key | Description |
|------------|----------|-------------|-------------|
| `q.datasets.process` | legacy-converter | `dataset.submitted` | Dataset files for processing |
| `q.lost-items.embed` | clip-service | `item.submitted` | Items for CLIP embedding |
| `q.lost-items.ingest` | qdrant-service | `item.embedded` | Items for Qdrant vectorization |
| `q.lost-items.publish` | c-publisher | `item.vectorized` | Items for dane.gov.pl |

## Supported File Formats

- **CSV** - Multiple items (one per row)
- **JSON** - Single or multiple items (array)
- **XML** - Single or multiple items
- **PDF** - Single item (extracted text)
- **DOCX** - Single item (extracted text)
- **HTML** - Single item (extracted text)
- **TXT** - Single item (plain text)

## Example: CSV Processing

### Input CSV (lost_items.csv):
```csv
Przedmiot,Opis,Kategoria,Miejsce,Data znalezienia,Email,Telefon
Portfel męski,Czarny portfel,portfele,ul. Marszałkowska 10,15.01.2024,biuro@urzad.pl,22-123-45-67
Telefon Samsung,Galaxy S21,telefony,Dworzec Centralny,14.01.2024,dworzec@pkp.pl,22-987-65-43
```

### Message to q.datasets.process:
```json
{
  "dataset_id": "batch-2024-01",
  "file_data": "UHJ6ZWRtaW90LE9waXMsS2F0ZWdvcmlhLE1pZWpzY2Us...",
  "file_name": "lost_items.csv",
  "file_format": ".csv"
}
```

### Result:
Service publishes **2 separate messages** to `item.submitted`:

**Message 1:**
```json
{
  "item_id": "550e8400-e29b-41d4-a716-446655440001",
  "text": "Portfel męski",
  "description": "Czarny portfel",
  "category": "portfele",
  "location": "ul. Marszałkowska 10",
  "date_lost": "2024-01-15T00:00:00",
  "contact_email": "biuro@urzad.pl",
  "contact_phone": "22-123-45-67",
  "timestamp": "2024-01-16T10:30:00"
}
```

**Message 2:**
```json
{
  "item_id": "550e8400-e29b-41d4-a716-446655440002",
  "text": "Telefon Samsung",
  "description": "Galaxy S21",
  "category": "telefony",
  "location": "Dworzec Centralny",
  "date_lost": "2024-01-14T00:00:00",
  "contact_email": "dworzec@pkp.pl",
  "contact_phone": "22-987-65-43",
  "timestamp": "2024-01-16T10:30:01"
}
```

Both items then flow through the normal processing pipeline:
1. CLIP service generates embeddings
2. Qdrant service stores vectors
3. Publisher service publishes to dane.gov.pl

## Integration with Gateway

The gateway service should publish messages to `q.datasets.process` when a dataset file is uploaded:

```go
// Example: Gateway publishing dataset for processing
func publishDatasetForProcessing(datasetID string, fileData []byte, fileName string) error {
    message := map[string]interface{}{
        "dataset_id": datasetID,
        "file_data": base64.StdEncoding.EncodeToString(fileData),
        "file_name": fileName,
        "file_format": filepath.Ext(fileName),
    }
    
    body, _ := json.Marshal(message)
    
    return channel.Publish(
        "lost-found.events",  // exchange
        "dataset.submitted",  // routing key
        false,               // mandatory
        false,               // immediate
        amqp.Publishing{
            DeliveryMode: amqp.Persistent,
            ContentType:  "application/json",
            Body:         body,
        },
    )
}
```

## Error Handling

### Successful Processing
- Message is acknowledged (`ACK`)
- All converted items are published

### Processing Error
- Message is negatively acknowledged with requeue (`NACK` with requeue=true)
- Service will retry processing

### Invalid Message
- Message is negatively acknowledged without requeue (`NACK` with requeue=false)
- Message is moved to dead letter queue (if configured)

## Monitoring

### Logs
```bash
# View service logs
docker logs -f odnalezione-legacy-converter

# Expected output:
# 📨 Received message: lost_items.csv
# 📄 Extracting text from lost_items.csv...
# 🔄 Converting to lost-items schema...
# Found 8 items in file
# 📤 Publishing 8 items to queue...
# ✅ Successfully published 8/8 items
```

### RabbitMQ Management UI
- Access: http://localhost:15674
- Check queue depths
- Monitor message rates
- View bindings

## Performance

- **Prefetch Count:** 1 (processes one file at a time)
- **Concurrent Processing:** Single consumer
- **File Size Limit:** Configured by gateway (no limit in converter)
- **Batch Size:** All items from one file published together

## Deployment

### Docker Compose
```bash
# Build and start
docker-compose build legacy-converter
docker-compose up -d legacy-converter

# Check status
docker-compose ps legacy-converter
docker logs odnalezione-legacy-converter
```

### Environment Variables
- `RABBITMQ_URL` - RabbitMQ connection string
- `RABBITMQ_EXCHANGE` - Exchange name (default: lost-found.events)
- `MINIO_ENDPOINT` - MinIO endpoint
- `MINIO_ACCESS_KEY` - MinIO access key
- `MINIO_SECRET_KEY` - MinIO secret key
- `MINIO_BUCKET_NAME` - MinIO bucket name

## Testing

### Manual Test with RabbitMQ CLI

```bash
# Encode a test file
base64 -w 0 examples/lost_items.csv > /tmp/file_data.txt

# Publish test message
docker exec odnalezione-rabbitmq rabbitmqadmin publish \
  exchange=lost-found.events \
  routing_key=dataset.submitted \
  payload="{\"dataset_id\":\"test-001\",\"file_data\":\"$(cat /tmp/file_data.txt)\",\"file_name\":\"lost_items.csv\",\"file_format\":\".csv\"}"

# Check logs
docker logs -f odnalezione-legacy-converter
```

### Python Test Script

```python
import pika
import json
import base64

# Read file
with open('examples/lost_items.csv', 'rb') as f:
    file_data = base64.b64encode(f.read()).decode('utf-8')

# Create message
message = {
    'dataset_id': 'test-001',
    'file_data': file_data,
    'file_name': 'lost_items.csv',
    'file_format': '.csv'
}

# Connect to RabbitMQ
connection = pika.BlockingConnection(
    pika.URLParameters('amqp://admin:admin123@localhost:5674/')
)
channel = connection.channel()

# Publish message
channel.basic_publish(
    exchange='lost-found.events',
    routing_key='dataset.submitted',
    body=json.dumps(message)
)

print("Message published!")
connection.close()
```

## Troubleshooting

### Service not consuming messages
- Check RabbitMQ connection in logs
- Verify queue exists: `docker exec odnalezione-rabbitmq rabbitmqctl list_queues`
- Check queue bindings: `docker exec odnalezione-rabbitmq rabbitmqctl list_bindings`

### Items not being published
- Check if extraction succeeded in logs
- Verify exchange and routing key are correct
- Check CLIP service is consuming from `q.lost-items.embed`

### Invalid message format
- Ensure file_data is base64 encoded
- Verify JSON structure matches expected format
- Check logs for JSON decode errors

## Migration from REST API

The REST API endpoints have been removed. If you need to process files:

1. **Option 1:** Use gateway service to upload and publish
2. **Option 2:** Publish directly to RabbitMQ (see testing examples)
3. **Option 3:** Create a separate upload service that publishes to the queue

## Benefits of RabbitMQ Integration

✅ **Decoupled Architecture** - Services don't need direct connections
✅ **Automatic Retry** - Failed messages can be requeued
✅ **Load Balancing** - Multiple consumers can process in parallel
✅ **Reliable Delivery** - Persistent messages survive restarts
✅ **Event-Driven** - Real-time processing as files arrive
✅ **Scalable** - Easy to add more consumers for high load

## Next Steps

1. Update gateway service to publish dataset files
2. Test with example CSV files
3. Monitor queue depths and processing times
4. Configure dead letter queues for failed messages
5. Add metrics and alerting
6. Consider adding multiple consumers for parallel processing
