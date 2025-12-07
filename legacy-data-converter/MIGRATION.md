# Legacy Data Converter - Migration to RabbitMQ Consumer

## Summary of Changes

The legacy-data-converter service has been completely refactored from a REST API service to a **RabbitMQ consumer** that integrates directly into the event-driven processing pipeline.

## Key Changes

### 1. **Architecture Change**

**Before:**
- Flask REST API with HTTP endpoints
- Standalone service on port 8083
- Manual file uploads via HTTP POST

**After:**
- RabbitMQ consumer service
- No HTTP port (runs as background worker)
- Automatic processing from queue
- Each item flows through entire pipeline

### 2. **New Components**

#### `rabbitmq_handler.py`
- Manages RabbitMQ connections
- Consumes from `q.datasets.process` queue
- Publishes converted items with routing key `item.submitted`
- Handles message acknowledgment and retry logic

#### `minio_handler.py`
- MinIO client for potential image handling
- Ready for future file storage needs

#### `test_publish.py`
- Test script to publish files to RabbitMQ
- Encodes files as base64
- Demonstrates message format

### 3. **Modified Components**

#### `main.py`
- **Removed:** All Flask routes (`/convert`, `/convert/batch`, `/extract`, `/health`, `/categories`)
- **Added:** `process_message()` callback for RabbitMQ
- **Changed:** Startup logic now calls `rabbitmq_handler.start_consuming()`
- **Enhanced:** Handles multiple items from single file (e.g., CSV with multiple rows)

#### `Dockerfile`
- **Removed:** Flask and Werkzeug
- **Removed:** Port exposure (8083)
- **Removed:** Health check endpoint
- **Added:** pika (RabbitMQ client)
- **Added:** minio (MinIO client)
- **Added:** New handler files

#### `docker-compose.yml`
- **Removed:** Port mapping (8083:8083)
- **Removed:** Volumes (legacy_uploads, legacy_output)
- **Added:** RabbitMQ environment variables
- **Added:** MinIO environment variables
- **Added:** Dependency on RabbitMQ and MinIO services

#### `requirements.txt` / `requirements-basic.txt`
- **Removed:** Flask==3.0.0, Werkzeug==3.0.1
- **Added:** pika==1.3.2 (RabbitMQ client)
- **Added:** minio==7.2.0 (MinIO client)

### 4. **Processing Flow**

#### Input Message (q.datasets.process)
```json
{
  "dataset_id": "uuid-v4",
  "file_data": "base64-encoded-content",
  "file_name": "lost_items.csv",
  "file_format": ".csv"
}
```

#### Processing Steps
1. Decode base64 file data
2. Save to temporary file
3. Extract text using `text_extractor.py`
4. Convert to lost-items schema using `nlp_converter.py`
5. **For each item:** Publish to `item.submitted`

#### Output Messages (item.submitted)
```json
{
  "item_id": "uuid-v4",
  "text": "Item title",
  "description": "Item description",
  "category": "kategoria",
  "location": "Location",
  "date_lost": "2024-01-15T00:00:00",
  "contact_email": "email@example.com",
  "contact_phone": "123456789",
  "timestamp": "2024-01-16T10:30:00"
}
```

### 5. **Multiple Items Handling**

**CSV with multiple rows:**
- Each row becomes a separate item
- Each item published individually to the queue
- All items enter the processing pipeline

**Example:**
```csv
Item 1
Item 2
Item 3
```
Results in **3 separate messages** to `item.submitted`, each processed by CLIP → Qdrant → Publisher.

### 6. **Event Flow Integration**

```
User/Gateway
    ↓
[dataset.submitted] → q.datasets.process
    ↓
Legacy Converter (extracts & converts)
    ↓
[item.submitted] → q.lost-items.embed
    ↓
CLIP Service (generates embeddings)
    ↓
[item.embedded] → q.lost-items.ingest
    ↓
Qdrant Service (stores vectors)
    ↓
[item.vectorized] → q.lost-items.publish
    ↓
Publisher Service (publishes to dane.gov.pl)
```

### 7. **RabbitMQ Configuration**

Queue `q.datasets.process` already exists (configured in `rabbitmq-init.sh`):
- **Binding:** `lost-found.events` exchange with routing key `dataset.submitted`
- **Durable:** Yes
- **Consumer:** legacy-converter

### 8. **Testing**

**Old way (removed):**
```bash
curl -X POST http://localhost:8083/convert -F "file=@file.csv"
```

**New way:**
```bash
python test_publish.py examples/lost_items.csv test-001
```

Or publish directly to RabbitMQ using pika.

### 9. **Monitoring**

**Check logs:**
```bash
docker logs -f odnalezione-legacy-converter
```

**Expected output:**
```
🚀 Starting legacy-data-converter service...
✅ Connected to RabbitMQ successfully
✅ Waiting for messages. To exit press CTRL+C
📨 Received message: lost_items.csv
📄 Extracting text from lost_items.csv...
🔄 Converting to lost-items schema...
Found 8 items in file
📤 Publishing 8 items to queue...
✅ Published item 550e8400-... to item.submitted
✅ Published item 661f9511-... to item.submitted
...
✅ Successfully published 8/8 items
```

**Check RabbitMQ:**
```bash
# List queues
docker exec odnalezione-rabbitmq rabbitmqctl list_queues

# Check queue depth
docker exec odnalezione-rabbitmq rabbitmqadmin list queues name messages
```

### 10. **Benefits**

✅ **Integrated Pipeline** - Items automatically flow through entire system
✅ **Scalable** - Can add multiple consumers for parallel processing
✅ **Reliable** - Message persistence and retry on failure
✅ **Decoupled** - No direct service-to-service dependencies
✅ **Automatic** - No manual intervention needed after file upload
✅ **Batch Processing** - CSV with 100 rows = 100 items automatically processed

### 11. **Files Changed**

**New files:**
- `rabbitmq_handler.py` - RabbitMQ consumer/publisher
- `minio_handler.py` - MinIO client wrapper
- `test_publish.py` - Test script for publishing
- `test-publish.ps1` - PowerShell test commands
- `RABBITMQ_INTEGRATION.md` - Integration documentation
- `MIGRATION.md` - This file

**Modified files:**
- `main.py` - Complete rewrite (Flask → RabbitMQ consumer)
- `Dockerfile` - Updated dependencies and removed port
- `docker-compose.yml` - Updated service configuration
- `requirements.txt` - Added pika, minio; removed Flask
- `requirements-basic.txt` - Added pika, minio; removed Flask
- `README.md` - Updated usage documentation

**Unchanged files:**
- `text_extractor.py` - Still extracts text from files
- `nlp_converter.py` - Still converts to lost-items schema
- `examples/` - Example files still work
- Tests - Still valid (unit tests for extractors)

### 12. **Breaking Changes**

⚠️ **REST API removed** - All HTTP endpoints no longer available:
- `/convert`
- `/convert/batch`
- `/extract`
- `/health`
- `/categories`

⚠️ **Port 8083 no longer exposed** - Service has no HTTP interface

⚠️ **Direct file upload not possible** - Must publish to RabbitMQ queue

### 13. **Migration Path**

If you were using the REST API:

1. **Option 1:** Update gateway to publish to `q.datasets.process`
2. **Option 2:** Use `test_publish.py` script for manual uploads
3. **Option 3:** Create adapter service that accepts HTTP and publishes to queue

### 14. **Environment Variables**

**New required:**
- `RABBITMQ_URL` - RabbitMQ connection string
- `RABBITMQ_EXCHANGE` - Exchange name (default: lost-found.events)
- `MINIO_ENDPOINT` - MinIO endpoint
- `MINIO_ACCESS_KEY` - MinIO access key
- `MINIO_SECRET_KEY` - MinIO secret key
- `MINIO_BUCKET_NAME` - Bucket name

**Removed:**
- `PORT` - No longer needed (no HTTP server)
- `UPLOAD_FOLDER` - No longer needed (temp files)
- `OUTPUT_FOLDER` - No longer needed (no file output)
- `MAX_FILE_SIZE` - Handled by gateway
- `DEBUG` - No longer needed

### 15. **Deployment**

**Build and start:**
```bash
docker-compose build legacy-converter
docker-compose up -d legacy-converter
```

**Verify running:**
```bash
docker ps | grep legacy-converter
docker logs odnalezione-legacy-converter
```

**Test with example:**
```bash
cd legacy-data-converter
python test_publish.py examples/lost_items.csv
```

## Conclusion

The service is now fully integrated into the event-driven architecture and processes files automatically from the RabbitMQ queue. Each converted item flows through the complete pipeline (CLIP embedding → Qdrant vectorization → dane.gov.pl publication).

The change enables:
- Automated batch processing
- Scalable architecture
- Reliable message delivery
- Full integration with existing services
