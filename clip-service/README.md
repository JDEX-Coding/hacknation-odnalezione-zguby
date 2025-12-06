# 🎨 CLIP Service

Python service that processes lost items using CLIP (Contrastive Language-Image Pre-training) model to generate embeddings for both text and images.

## 📋 Overview

The CLIP service is part of the lost-and-found system that:
- Consumes messages from RabbitMQ (`q.lost-items.embed` queue)
- Downloads images from MinIO object storage
- Generates embeddings using OpenAI's CLIP model for:
  - Text descriptions (title, description, category)
  - Images (if MinIO key provided)
- Publishes embeddings via RabbitMQ for storage by the Qdrant service
- Communicates exclusively through RabbitMQ (no direct database access)
- Supports backward compatibility with legacy image URL format

## 🏗️ Architecture

```
┌─────────────┐     item.submitted     ┌──────────────┐
│   Gateway   │ ────────────────────►  │  RabbitMQ    │
└─────────────┘                        └──────┬───────┘
                                              │
                                              ▼
                     ┌──────────┐   ┌─────────────────┐
                     │  MinIO   │◄──│  CLIP Service   │
                     │  Object  │   │  - Download img │
                     │  Storage │   │  - Generate     │
                     └──────────┘   │    embeddings   │
                                    └────────┬────────┘
                                             │
                                             ▼
                                    ┌───────────────┐
                                    │   RabbitMQ    │
                                    │item.embedded  │
                                    │+ embedding [] │
                                    └───────┬───────┘
                                            │
                                            ▼
                                    ┌──────────────┐
                                    │Qdrant Service│
                                    │ (Go service) │
                                    └──────────────┘
```

## 🔧 Components

### `main.py`
Main service orchestrator that:
- Connects to RabbitMQ
- Consumes messages from `q.lost-items.embed` (routing key: `item.submitted`)
- Coordinates the processing pipeline
- Publishes results to `q.lost-items.ingest` (routing key: `item.embedded`)

### `clip_handler.py`
CLIP model handler that:
- Loads the CLIP model (openai/clip-vit-base-patch32)
- Encodes text into embeddings (512 dimensions)
- Encodes images into embeddings (512 dimensions)
- Normalizes embeddings for cosine similarity

### `minio_handler.py`
MinIO object storage handler that:
- Downloads images from MinIO S3-compatible storage
- Validates image files
- Manages temporary storage
- Cleans up after processing
- Supports both file and in-memory image loading

### `message_converter.py`
Message format converter that:
- Normalizes incoming messages to consistent format
- Converts legacy `image_url` to `image_key` format
- Validates message structure and content
- Tracks format usage statistics
- Ensures backward compatibility

## 🚀 Running the Service

### With Docker Compose (Recommended)

The service is integrated into the main docker-compose.yml:

```bash
docker-compose up clip-service
```

### Standalone Development

1. **Install dependencies:**
```bash
pip install -r requirements.txt
```

2. **Configure environment:**
```bash
# Copy the example .env file
cp .env.example .env

# Edit .env file with your local settings
# For local development, use localhost:9000 for MinIO
```

The `.env` file should contain:
```bash
RABBITMQ_URL=amqp://admin:admin123@localhost:5672/
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin123
MINIO_BUCKET_NAME=lost-items-images
```

**Note**: In Docker, these variables are set in `docker-compose.yml` and use service names (e.g., `minio:9000` instead of `localhost:9000`).

3. **Run the service:**
```bash
python main.py
```

## 📨 Message Format

### Input Message (`item.submitted`)

Consumed from queue: `q.lost-items.embed`

**Preferred Format (MinIO):**
```json
{
  "item_id": "550e8400-e29b-41d4-a716-446655440000",
  "text": "Lost Blue Backpack",
  "description": "Contains laptop and important documents",
  "category": "Bags",
  "location": "Central Train Station",
  "date_lost": "2024-12-06T10:30:00Z",
  "image_key": "items/backpack_550e8400.jpg",
  "contact_info": "john@example.com"
}
```

**Legacy Format (Backward Compatible):**
```json
{
  "item_id": "550e8400-e29b-41d4-a716-446655440000",
  "text": "Lost Blue Backpack",
  "description": "Contains laptop and important documents",
  "category": "Bags",
  "location": "Central Train Station",
  "date_lost": "2024-12-06T10:30:00Z",
  "image_url": "https://example.com/images/backpack.jpg",
  "contact_info": "john@example.com"
}
```

**Note**: The service automatically converts `image_url` to `image_key` format for backward compatibility, but using `image_key` is recommended.

### Output Message (`item.embedded`)

Published to exchange with routing key: `item.embedded` (routed to `q.lost-items.ingest` for Qdrant Service)

```json
{
  "item_id": "550e8400-e29b-41d4-a716-446655440000",
  "embedding": [0.123, -0.456, 0.789, ... ],
  "title": "Lost Blue Backpack",
  "description": "Contains laptop and important documents",
  "category": "Bags",
  "location": "Central Train Station",
  "date_lost": "2024-12-06T10:30:00Z",
  "image_key": "items/backpack_550e8400.jpg",
  "contact_info": "john@example.com",
  "timestamp": "2024-12-06T10:35:22.123456",
  "has_image_embedding": true
}
```

**Note**: The `embedding` field contains the 512-dimensional CLIP embedding vector that will be stored in Qdrant by the qdrant-service.

## 🔑 Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `RABBITMQ_URL` | `amqp://admin:admin123@localhost:5672/` | RabbitMQ connection URL |
| `MINIO_ENDPOINT` | `minio:9000` | MinIO server endpoint |
| `MINIO_ACCESS_KEY` | `minioadmin` | MinIO access key |
| `MINIO_SECRET_KEY` | `minioadmin123` | MinIO secret key |
| `MINIO_BUCKET_NAME` | `lost-items-images` | MinIO bucket name for images |

## 🎯 Features

### Multi-modal Embeddings
- **Text Embeddings**: Combines title, description, and category
- **Image Embeddings**: Generated from downloaded images
- **Combined Embeddings**: Averages text and image embeddings when both available

### Robust Image Handling
- MinIO object storage integration
- Object validation and size checking
- File size limits (max 10MB)
- Image validation before processing
- Automatic cleanup of temporary files
- Support for legacy URL-based format

### Message Format Conversion
- **Automatic Format Detection**: Handles both `image_key` and `image_url` fields
- **Backward Compatibility**: Converts legacy URLs to MinIO keys automatically
- **Smart Conversion**: Extracts keys from MinIO URLs, creates standard paths for external URLs
- **Validation**: Ensures all messages meet required format standards
- **Statistics Tracking**: Monitors format usage and conversion metrics
- **Priority System**: `image_key` takes precedence when both fields present

### Error Handling
- Automatic retry for failed image downloads
- Message requeuing for transient failures
- Dead letter handling for permanent failures
- Comprehensive logging

### Performance
- QoS limiting (1 message at a time)
- Efficient batch processing capability
- Connection pooling and heartbeats
- Health checks

## 📊 Monitoring

### Logs
The service provides detailed logging:
- `INFO`: Normal operation events
- `ERROR`: Processing errors with stack traces
- `DEBUG`: Detailed debugging information

### Health Checks
Docker health check runs every 30 seconds to verify the service is responsive.

## 🧪 Testing

### Unit Tests

Run the message converter unit tests:

```bash
python -m unittest test_message_converter.py -v
```

All 17 tests should pass, covering:
- Format conversion (URL → MinIO key)
- Message validation
- Text normalization
- Object key validation
- Statistics tracking

### Test with Event Emulator

```bash
cd ../event-emulator
make run
```

This will send test messages to the queue that the CLIP service will process.

### Test Message Formats

Test both input formats to verify backward compatibility:

**Native MinIO format:**
```json
{
  "item_id": "test-001",
  "text": "Test item",
  "image_key": "items/test.jpg"
}
```

**Legacy URL format (auto-converted):**
```json
{
  "item_id": "test-002",
  "text": "Test item",
  "image_url": "https://example.com/test.jpg"
}
```

### Manual Testing

You can publish test messages directly to RabbitMQ:

```bash
# Using RabbitMQ Management UI at http://localhost:15672
# Navigate to Queues -> q.lost-items.embed -> Publish message
# Or publish to exchange 'lost-found.events' with routing key 'item.submitted'
```

## 🛠️ Development

### Project Structure
```
clip-service/
├── main.py                    # Main service entry point
├── clip_handler.py            # CLIP model operations
├── minio_handler.py           # MinIO object storage handler
├── message_converter.py       # Message format converter
├── image_downloader.py        # Legacy image downloader (deprecated)
├── test_message_converter.py  # Unit tests for message converter
├── requirements.txt           # Python dependencies
├── Dockerfile                 # Docker build configuration
├── Makefile                   # Development commands
├── README.md                  # This file
└── MINIO_MIGRATION.md         # MinIO migration guide
```

### Adding New Features

1. **New embedding models**: Modify `clip_handler.py` to support additional models
2. **Custom preprocessing**: Extend `minio_handler.py` for image preprocessing
3. **Message enrichment**: Add metadata to published messages in `main.py`
4. **Format converters**: Extend `message_converter.py` for new field transformations

### Migration from URL to MinIO

If you have existing systems using `image_url`:

1. **Upload images to MinIO** using the MinIO Console or SDK
2. **Update message producers** to use `image_key` instead of `image_url`
3. **Optional**: Keep using `image_url` temporarily - the service auto-converts

See [MINIO_MIGRATION.md](MINIO_MIGRATION.md) for detailed migration instructions.

## 📦 Dependencies

- **torch**: Deep learning framework
- **transformers**: HuggingFace transformers (CLIP model)
- **pillow**: Image processing
- **pika**: RabbitMQ client
- **minio**: MinIO Python SDK for object storage
- **requests**: HTTP client (legacy support)
- **numpy**: Numerical operations

## 🐛 Troubleshooting

### Service won't start
- Check RabbitMQ is running: `docker ps | grep rabbitmq`
- Verify RabbitMQ is healthy: `docker logs odnalezione-rabbitmq`
- Check logs: `docker logs odnalezione-clip-service`

### Images not processing
- Verify MinIO service is running: `docker ps | grep minio`
- Check MinIO bucket exists: Access MinIO Console at http://localhost:9001
- Verify image objects exist in MinIO
- Check network connectivity between services
- Review file size limits in `minio_handler.py`
- Check logs for format conversion warnings

### Low performance
- Increase prefetch count in `main.py`
- Use GPU if available (modify Dockerfile)
- Consider batching operations

## 📄 License

See LICENSE file in the project root.

## 🤝 Contributing

This service is part of the Hacknation Lost & Found system. Follow the main project's contribution guidelines.
