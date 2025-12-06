# MinIO Migration Guide

## Overview

The CLIP service has been updated to use **MinIO object storage** instead of downloading images from external URLs. This provides better security, reliability, and performance.

## Changes Made

### 1. New MinIO Handler (`minio_handler.py`)
- Replaces the `image_downloader.py` module
- Downloads images from MinIO S3-compatible storage
- Supports both file-based and in-memory image loading
- Includes validation, cleanup, and object management functions

### 2. Message Format Converter (`message_converter.py`)
- **Automatic Conversion**: Converts legacy `image_url` to `image_key` format
- **Validation**: Ensures messages meet required format standards
- **Normalization**: Cleans and standardizes all text fields
- **Statistics**: Tracks format usage and conversion metrics
- **Backward Compatibility**: Supports both old and new message formats seamlessly

### 3. Updated Dependencies
- Added `minio>=7.2.0` to `requirements.txt`

### 4. Docker Compose Updates
- Added MinIO environment variables to `clip-service`:
  - `MINIO_ENDPOINT`: MinIO server endpoint
  - `MINIO_ACCESS_KEY`: Access credentials
  - `MINIO_SECRET_KEY`: Secret credentials
  - `MINIO_BUCKET_NAME`: Bucket name for images
- Added dependency on MinIO service health check

### 5. Message Format Change
**Preferred Format (New):** Messages use `image_key` field with MinIO object keys
```json
{
  "item_id": "item123",
  "text": "Lost black backpack",
  "image_key": "items/item123.jpg"
}
```

**Legacy Format (Auto-Converted):** Messages with `image_url` are automatically converted
```json
{
  "item_id": "item123",
  "text": "Lost black backpack",
  "image_url": "https://example.com/images/backpack.jpg"
}
```
↓ *Automatically converts to* ↓
```json
{
  "image_key": "items/backpack.jpg"
}
```

**Note**: The service logs warnings when legacy format is detected, encouraging migration to `image_key`.

## Usage

### Environment Variables
```bash
MINIO_ENDPOINT=minio:9000          # MinIO server address
MINIO_ACCESS_KEY=minioadmin        # MinIO access key
MINIO_SECRET_KEY=minioadmin123     # MinIO secret key
MINIO_BUCKET_NAME=lost-items-images # Bucket containing images
```

### Uploading Images to MinIO

Before sending messages to the CLIP service, images must be uploaded to MinIO:

```python
from minio import Minio

client = Minio(
    "localhost:9000",
    access_key="minioadmin",
    secret_key="minioadmin123",
    secure=False
)

# Upload image
client.fput_object(
    "lost-items-images",
    "items/item123.jpg",  # Object key
    "/path/to/local/image.jpg"
)
```

### MinIO Handler API

```python
from minio_handler import MinIOHandler

handler = MinIOHandler()

# Download image to file
image_path = handler.download_image("items/item123.jpg", "item123")

# Download image to memory (PIL Image)
image = handler.download_image_to_memory("items/item123.jpg")

# Check if object exists
exists = handler.object_exists("items/item123.jpg")

# List objects
objects = handler.list_objects(prefix="items/")

# Get object info
info = handler.get_image_info("items/item123.jpg")
```

## Benefits

1. **Security**: Images stored securely in private object storage
2. **Reliability**: No external HTTP requests, controlled environment
3. **Performance**: Fast local network access, no internet latency
4. **Consistency**: S3-compatible storage with versioning support
5. **Scalability**: MinIO can be clustered for high availability
6. **Backward Compatibility**: Seamless transition with automatic format conversion
7. **Validation**: Built-in message validation and normalizationt works
2. **Monitor logs** - Check for conversion warnings in service logs
3. **Upload to MinIO** - Gradually move images to MinIO bucket
4. **Update producers** - Switch to `image_key` format when ready
5. **Verify** - Monitor statistics via logs

### Option 2: Immediate Migration

For new deployments or complete migration:

1. **Upload images to MinIO bucket** using Console UI or SDK
2. **Update message producers** to use `image_key` instead of `image_url`
3. **Ensure MinIO service is running** and healthy
4. **Rebuild and restart** the clip-service container
5. **Test** with sample messages
## Migration Steps

If you have existing workflows using URLs:

1. Upload images to MinIO bucket
2. Update message producers to use `image_key` instead of `image_url`
3. Ensure MinIO service is running and healthy
4. Rebuild and restart the clip-service container

## Testing

Access MinIO Console UI at http://localhost:9001
- Username: `minioadmin`
- Password: `minioadmin123`

You can browse, upload, and manage images through the web interface.
