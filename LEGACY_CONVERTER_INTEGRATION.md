# Legacy Data Converter Integration

## New Service Added: legacy-data-converter

A Python microservice for extracting text from various file formats and converting them to the lost-items schema using NLP.

## Service Architecture Update

The system now includes 7 services:

1. **clip-service** (Port: internal) - Image embedding
2. **qdrant-service** (Port: 8081) - Vector search
3. **qdrant-db** (Ports: 6333, 6334) - Vector database
4. **rabbitmq** (Ports: 5674, 15674) - Message broker
5. **minio** (Ports: 9000, 9001) - Object storage
6. **postgres** (Port: 5434) - Relational database
7. **a-gateway** (Port: 8082) - Main API gateway
8. **c-publisher** (Port: internal) - dane.gov.pl publisher
9. **legacy-converter** (Port: 8083) - NEW: Legacy data conversion

## Legacy Converter Features

### Supported File Formats
- PDF documents
- DOCX (Microsoft Word)
- HTML files
- Plain text (TXT)
- JSON documents
- XML files
- CSV spreadsheets

### API Endpoints

- `GET /health` - Health check
- `POST /convert` - Convert single file
- `POST /convert/batch` - Convert multiple files
- `POST /extract` - Extract text only
- `GET /categories` - List categories

### Access

**Local**: http://localhost:8083
**Container**: http://legacy-converter:8083 (internal network)

## Integration Workflow

### Scenario 1: Import Legacy CSV Data

```bash
# 1. Convert CSV file
curl -X POST http://localhost:8083/convert \
  -F "file=@legacy_data.csv" \
  -F "dataset_id=migration-2024" \
  -o converted.json

# 2. Parse and submit to gateway
# Process converted.json and POST each item to gateway API
```

### Scenario 2: Batch Document Processing

```bash
# Convert multiple documents at once
curl -X POST http://localhost:8083/convert/batch \
  -F "files=@doc1.pdf" \
  -F "files=@doc2.docx" \
  -F "files=@data.csv"
```

## Docker Compose Configuration

Added to `docker-compose.yml`:

```yaml
legacy-converter:
  container_name: odnalezione-legacy-converter
  build:
    context: ./legacy-data-converter
    dockerfile: Dockerfile
  ports:
    - "8083:8083"
  environment:
    - PORT=8083
    - UPLOAD_FOLDER=/app/uploads
    - OUTPUT_FOLDER=/app/output
    - MAX_FILE_SIZE=10485760
    - DEBUG=false
  volumes:
    - legacy_uploads:/app/uploads
    - legacy_output:/app/output
  restart: unless-stopped
  networks:
    - odnalezione-network
```

## Volume Changes

Added two new volumes:
- `legacy_uploads` - Temporary file uploads
- `legacy_output` - Converted JSON files

## Ports Summary

| Service | Port(s) | Purpose |
|---------|---------|---------|
| RabbitMQ | 5674, 15674 | Message broker, Management UI |
| Qdrant | 6333, 6334 | Vector DB HTTP, gRPC |
| Qdrant Service | 8081 | Service API |
| MinIO | 9000, 9001 | Object storage, Console |
| PostgreSQL | 5434 | Database |
| Gateway | 8082 | Main API |
| **Legacy Converter** | **8083** | **File conversion** |

## Network Architecture

All services are on the `odnalezione-network` bridge network:
- Internal communication via service names
- External access via published ports

## Usage

### Start All Services
```bash
docker-compose up -d
```

### Start Only Legacy Converter
```bash
docker-compose up legacy-converter
```

### Check Service Health
```bash
curl http://localhost:8083/health
```

### View Logs
```bash
docker logs odnalezione-legacy-converter
```

## Development Workflow

1. **Place files in examples folder**
2. **Test conversion locally**
   ```bash
   python nlp_converter.py examples/lost_items.csv
   ```
3. **Build and test in Docker**
   ```bash
   docker-compose build legacy-converter
   docker-compose up legacy-converter
   ```
4. **Test API endpoints**
   ```bash
   curl -X POST http://localhost:8083/convert -F "file=@examples/lost_items.csv"
   ```

## Data Flow

```
Legacy Files → Legacy Converter → Lost Items JSON
     ↓                                    ↓
  Upload                            Gateway API
     ↓                                    ↓
  Convert                         Database Insert
     ↓                                    ↓
  Output                          CLIP Processing
     ↓                                    ↓
  JSON File                       Qdrant Vectorization
                                          ↓
                                    dane.gov.pl
```

## Testing

### Test with provided examples:
```bash
cd legacy-data-converter/examples

# Test CSV
curl -X POST http://localhost:8083/convert -F "file=@lost_items.csv"

# Test JSON
curl -X POST http://localhost:8083/convert -F "file=@items.json"

# Test XML
curl -X POST http://localhost:8083/convert -F "file=@items.xml"

# Test TXT
curl -X POST http://localhost:8083/convert -F "file=@report.txt"
```

## Documentation

All documentation is in the `legacy-data-converter/` folder:

- **README.md** - Full documentation
- **QUICKSTART.md** - Quick start guide
- **SERVICE_SUMMARY.md** - Technical summary
- **examples/README.md** - Example files documentation

## Next Steps

1. Test the service with real legacy data
2. Integrate with gateway for automatic processing
3. Create automation scripts for batch imports
4. Monitor performance and adjust MAX_FILE_SIZE if needed
5. Add custom category keywords if needed
6. Consider enabling full NLP features (spaCy, transformers)

## Maintenance

### Clean up output files
```bash
docker exec odnalezione-legacy-converter rm -rf /app/output/*
```

### Rebuild after changes
```bash
docker-compose build legacy-converter
docker-compose restart legacy-converter
```

### Run tests
```bash
cd legacy-data-converter
pytest -v
```

## Environment Variables

Can be customized in docker-compose.yml:

- `PORT` - Service port (default: 8083)
- `UPLOAD_FOLDER` - Upload directory
- `OUTPUT_FOLDER` - Output directory
- `MAX_FILE_SIZE` - Max file size in bytes (default: 10MB)
- `DEBUG` - Enable debug mode (default: false)

## Security Notes

- Files are validated by extension (whitelist)
- Filenames are sanitized
- Size limits enforced
- Temporary files cleaned after processing
- No code execution from uploads

---

**Status**: Service ready for testing ✅
**Integration**: Complete ✅
**Documentation**: Complete ✅
