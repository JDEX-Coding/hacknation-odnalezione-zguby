# Dataset Publication Pipeline

## Overview

This pipeline allows publishing datasets from the PostgreSQL database to dane.gov.pl using RabbitMQ for asynchronous processing.

## Architecture

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐      ┌───────────────┐
│   Gateway   │─────▶│  PostgreSQL  │      │  RabbitMQ   │─────▶│  Publisher    │
│  (Service A)│      │   Database   │      │             │      │  (Service C)  │
└─────────────┘      └──────────────┘      └─────────────┘      └───────────────┘
      │                     │                      │                     │
      │  1. Fetch Dataset   │                      │                     │
      │◀────────────────────┘                      │                     │
      │                                            │                     │
      │  2. Publish Event                          │                     │
      │───────────────────────────────────────────▶│                     │
      │         (dataset.publish)                  │                     │
      │                                            │  3. Consume Event   │
      │                                            │────────────────────▶│
      │                                            │                     │
      │                                            │                     ▼
      │                                            │            ┌─────────────────┐
      │                                            │            │  dane.gov.pl    │
      │                                            │            │   API Client    │
      │                                            │            └─────────────────┘
      │                                            │                     │
      │                                            │  4. Success Event   │
      │                                            │◀────────────────────┘
      │                                            │   (dataset.published)
```

## Event Flow

### Event: `dataset.publish`

**Queue:** `q.datasets.publish`  
**Producer:** Service A (Gateway)  
**Consumer:** Service C (Publisher)

**Payload:**
```json
{
  "dataset_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Rejestr rzeczy znalezionych 2025",
  "notes": "Baza danych zgłoszonych rzeczy znalezionych w 2025 roku",
  "url": "https://example.com/dataset-url",
  "institution_name": "Urząd Miasta",
  "email": "kontakt@urzad.pl",
  "categories": ["transport", "inne"],
  "tags": ["rzeczy znalezione", "2025"],
  "timestamp": "2025-12-06T10:40:00Z"
}
```

### Event: `dataset.published`

**Producer:** Service C (Publisher)  
**Payload:**
```json
{
  "dataset_id": "550e8400-e29b-41d4-a716-446655440000",
  "dane_gov_id": "abc-123-def",
  "published_at": "2025-12-06T10:41:00Z",
  "dane_gov_url": "http://localhost:8000/api/v1/datasets/abc-123-def",
  "publication_date": "2025-12-06T10:41:00Z"
}
```

## API Endpoints

### Gateway (Service A)

#### List Datasets
```http
GET /api/datasets
```

**Response:**
```json
{
  "success": true,
  "datasets": [...],
  "count": 5
}
```

#### Get Dataset
```http
GET /api/datasets/{id}
```

**Response:**
```json
{
  "success": true,
  "dataset": {
    "id": "...",
    "title": "...",
    "notes": "...",
    ...
  }
}
```

#### Get Dataset with Items
```http
GET /api/datasets/{id}/items
```

**Response:**
```json
{
  "success": true,
  "dataset": {
    "id": "...",
    "title": "...",
    "items": [...]
  }
}
```

#### Publish Dataset
```http
POST /api/datasets/{id}/publish
```

**Response:**
```json
{
  "success": true,
  "dataset_id": "...",
  "message": "Dataset queued for publication to dane.gov.pl",
  "status": "pending"
}
```

## Implementation Details

### Gateway (Service A)

**New Files:**
- `internal/handlers/dataset_publish.go` - Dataset publication handlers

**Key Functions:**
- `PublishDatasetHandler` - Publishes dataset to RabbitMQ
- `ListDatasetsAPIHandler` - Lists all datasets
- `GetDatasetAPIHandler` - Gets single dataset
- `GetDatasetWithItemsAPIHandler` - Gets dataset with items

### Publisher (Service C)

**New Files:**
- `internal/models/dataset_events.go` - Dataset event models

**Updated Files:**
- `main.go` - Added dataset consumer alongside item consumer
- `internal/consumer/rabbitmq_consumer.go` - Added `ConsumeDatasets` method

**Key Changes:**
- Dual consumer setup (items + datasets)
- Dataset creation via `CreateDataset` API
- Success event publishing

## Testing

### Manual Test

1. **Start services:**
```bash
docker compose up
```

2. **Create a dataset** via web UI:
   - Go to http://localhost:8082/create-dataset
   - Fill in dataset details
   - Upload a document

3. **Publish dataset** via API:
```powershell
.\test-dataset-pipeline.ps1
```

### Test Script Usage

```powershell
# Test with automatic dataset selection
.\test-dataset-pipeline.ps1

# Test with specific dataset
.\test-dataset-pipeline.ps1 -DatasetID "your-dataset-id"

# Test with different gateway URL
.\test-dataset-pipeline.ps1 -GatewayUrl "http://localhost:8080"
```

## Monitoring

### RabbitMQ Management UI
- URL: http://localhost:15674
- Username: `admin`
- Password: `admin123`
- Check queues: `q.datasets.publish`, `q.lost-items.publish`

### Publisher Logs
```bash
docker logs odnalezione-publisher -f
```

### Gateway Logs
```bash
docker logs odnalezione-gateway -f
```

## Configuration

### Gateway Environment Variables

No additional configuration needed - uses existing RabbitMQ connection.

### Publisher Environment Variables

```env
# Existing variables work for both item and dataset publishing
RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/
RABBITMQ_EXCHANGE=lost-found.events
DANE_GOV_API_URL=http://localhost:8000
DANE_GOV_EMAIL=admin2@mcod.local
DANE_GOV_PASSWORD=Hacknation-2025
```

## Future Enhancements

1. **Batch Publishing** - Publish multiple datasets at once
2. **Status Tracking** - Store publication status in database
3. **Retry Logic** - Automatic retry on API failures
4. **Dataset Updates** - Handle dataset modifications
5. **Resource Publishing** - Publish dataset resources (items) to dane.gov.pl
6. **Webhooks** - Notify on publication success/failure

## Troubleshooting

### Dataset Not Found
- Ensure dataset exists in database
- Check dataset ID is correct
- Verify database connection

### RabbitMQ Connection Failed
- Check RabbitMQ is running: `docker ps`
- Verify credentials in environment variables
- Check queue exists in RabbitMQ management UI

### dane.gov.pl API Error
- Verify mock API is running: `docker logs odnalezione-mock-api`
- Check API credentials are correct
- Review API response in publisher logs

### Event Not Consumed
- Check consumer is running: `docker logs odnalezione-publisher`
- Verify routing key matches: `dataset.publish`
- Check queue binding in RabbitMQ UI

## Related Documentation

- [PAYLOADS.md](./PAYLOADS.md) - Complete event payload specifications
- [DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md) - Database schema with dataset tables
- [service-c-publisher/README.md](./service-c-publisher/README.md) - Publisher service details
- [service-a-gateway/README.md](./service-a-gateway/README.md) - Gateway service details
