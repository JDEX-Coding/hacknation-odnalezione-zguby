# Legacy Data Converter Service - Summary

## Overview

The **legacy-data-converter** service is a Python-based microservice designed to extract text from various file formats (PDF, DOCX, HTML, TXT, JSON, XML, CSV) and convert them to the lost-items schema using NLP techniques.

## Service Details

- **Port**: 8083
- **Technology**: Python 3.11 + Flask
- **Container**: `odnalezione-legacy-converter`
- **API Type**: REST API

## Architecture

### Components

1. **text_extractor.py**
   - Handles file parsing and text extraction
   - Supports 7 file formats
   - Returns structured data with text, metadata, and raw data

2. **nlp_converter.py**
   - Converts extracted text to lost-items schema
   - Uses pattern matching, keyword detection, and optional NLP
   - Supports Polish language
   - Handles both structured and unstructured data

3. **main.py**
   - Flask REST API server
   - Provides endpoints for conversion and batch processing
   - Manages file uploads and outputs

### Supported File Formats

| Format | Extension | Features |
|--------|-----------|----------|
| PDF | .pdf | Multi-page text extraction |
| Word | .docx | Paragraphs and tables |
| HTML | .html, .htm | Clean text extraction |
| Text | .txt | Multiple encoding support |
| JSON | .json | Direct field mapping |
| XML | .xml | Nested structure parsing |
| CSV | .csv | Row-based extraction |

## API Endpoints

### `GET /health`
Health check endpoint

### `POST /convert`
Convert single file to lost-item schema
- **Input**: multipart/form-data with `file` and optional `dataset_id`
- **Output**: JSON with converted item

### `POST /convert/batch`
Convert multiple files
- **Input**: multipart/form-data with `files[]` and optional `dataset_id`
- **Output**: JSON with array of converted items

### `POST /extract`
Extract text only (no conversion)
- **Input**: multipart/form-data with `file`
- **Output**: JSON with extracted text and metadata

### `GET /categories`
Get list of supported categories
- **Output**: JSON array of categories

## Lost-Items Schema Output

```json
{
  "id": "uuid",
  "title": "string",
  "description": "string",
  "category": "string",
  "location": "string",
  "found_date": "ISO 8601 datetime",
  "reporting_date": "ISO 8601 datetime",
  "reporting_location": "string",
  "contact_email": "string",
  "contact_phone": "string",
  "status": "pending",
  "created_at": "ISO 8601 datetime",
  "updated_at": "ISO 8601 datetime",
  "metadata": {
    "source_file": "string",
    "source_format": "string",
    "conversion_timestamp": "ISO 8601 datetime"
  }
}
```

## Categories

The service recognizes 11 categories:
- dokumenty (documents)
- elektronika (electronics)
- biżuteria (jewelry)
- odzież (clothing)
- torby i plecaki (bags and backpacks)
- klucze (keys)
- portfele (wallets)
- telefony (phones)
- zwierzęta (animals/pets)
- pojazdy (vehicles)
- inne (other)

## NLP Features

### Automatic Extraction

1. **Category Detection**: Keyword-based classification
2. **Location Extraction**: Pattern matching for Polish addresses
3. **Date Parsing**: Multiple date formats (DD.MM.YYYY, YYYY-MM-DD, Polish month names)
4. **Contact Extraction**: Email and phone number patterns
5. **Title Generation**: From first significant line
6. **Description**: Full text with length limiting

### Structured Data Mapping

For CSV, JSON, XML files, the converter attempts to map fields:
- title → tytuł, nazwa, przedmiot, name, item
- description → opis, desc, details, szczegóły
- category → kategoria, typ, type
- location → lokalizacja, miejsce, adres, address
- found_date → data_znalezienia, date, data, found
- contact_email → email, e-mail, kontakt
- contact_phone → telefon, tel, phone, phone_number

## Docker Integration

### Service Configuration in docker-compose.yml

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
  volumes:
    - legacy_uploads:/app/uploads
    - legacy_output:/app/output
  networks:
    - odnalezione-network
```

## File Structure

```
legacy-data-converter/
├── main.py                    # Flask API server
├── text_extractor.py          # Text extraction module
├── nlp_converter.py           # NLP conversion module
├── requirements.txt           # Full dependencies (with NLP)
├── requirements-basic.txt     # Basic dependencies
├── Dockerfile                 # Basic Docker image
├── Dockerfile.full            # Full Docker image with NLP
├── Makefile                   # Build and run commands
├── README.md                  # Full documentation
├── QUICKSTART.md             # Quick start guide
├── .gitignore                # Git ignore rules
├── test_text_extractor.py    # Unit tests for extractor
├── test_nlp_converter.py     # Unit tests for converter
├── uploads/                   # Temporary upload folder
├── output/                    # Converted files output
└── examples/                  # Example test files
    ├── lost_items.csv
    ├── items.json
    ├── items.xml
    ├── report.txt
    └── README.md
```

## Dependencies

### Basic (requirements-basic.txt)
- Flask 3.0.0
- PyPDF2 3.0.1
- python-docx 1.1.0
- beautifulsoup4 4.12.2
- lxml 4.9.3
- python-dateutil 2.8.2

### Full NLP (requirements.txt)
All basic dependencies plus:
- spacy 3.7.2
- transformers 4.36.0
- torch 2.1.2

## Usage Examples

### Start Service
```bash
docker-compose up legacy-converter
```

### Convert Single File
```bash
curl -X POST http://localhost:8083/convert \
  -F "file=@document.pdf"
```

### Batch Conversion
```bash
curl -X POST http://localhost:8083/convert/batch \
  -F "files=@file1.csv" \
  -F "files=@file2.json" \
  -F "dataset_id=batch-001"
```

### Extract Text Only
```bash
curl -X POST http://localhost:8083/extract \
  -F "file=@document.pdf"
```

## Testing

Run unit tests:
```bash
cd legacy-data-converter
pip install pytest
pytest -v
```

Test with example files:
```bash
cd examples
curl -X POST http://localhost:8083/convert -F "file=@lost_items.csv"
```

## Integration Points

### With Gateway Service (service-a-gateway)
- Gateway can forward uploaded files to converter
- Converted items can be submitted back to gateway API
- Supports bulk import of legacy data

### With Database
- Converted items match PostgreSQL schema
- Can be directly inserted into `lost_items` table
- Supports `dataset_items` junction table

### With Processing Pipeline
- Converted items can enter normal processing flow
- CLIP service can process images (if provided)
- Qdrant service can vectorize items
- Publisher can publish to dane.gov.pl

## Performance Considerations

- **File Size Limit**: 10MB default (configurable)
- **Concurrent Uploads**: Flask handles multiple concurrent requests
- **Temporary Storage**: Files cleaned up after processing
- **Output Retention**: Converted JSON files saved to output folder

## Security Considerations

- File type validation (whitelist approach)
- Filename sanitization (secure_filename)
- Size limits enforced
- No code execution from uploaded files
- Temporary file cleanup

## Future Enhancements

Possible improvements:
1. **Image OCR**: Add OCR capability for scanned images
2. **Advanced NLP**: Fine-tuned models for better extraction
3. **Multi-language**: Support for other languages
4. **Batch Scheduling**: Automatic directory scanning
5. **Database Direct**: Direct database insertion option
6. **Webhook Support**: Notify other services on completion
7. **Progress Tracking**: Long-running batch job status
8. **Data Validation**: Schema validation before output

## Maintenance

### Logs
```bash
docker logs odnalezione-legacy-converter
```

### Clean Output
```bash
docker exec odnalezione-legacy-converter rm -rf /app/output/*
```

### Rebuild
```bash
docker-compose build legacy-converter
docker-compose up -d legacy-converter
```

## Support Files

- **README.md**: Comprehensive documentation
- **QUICKSTART.md**: Quick start and examples
- **examples/README.md**: Test file documentation
- **Makefile**: Common commands
- **tests**: Unit tests for components

## Success Metrics

The service is working correctly when:
- Health endpoint returns 200 OK
- CSV files extract all rows as separate items
- JSON/XML files map fields correctly
- Text files extract contact information
- Categories are detected accurately
- Dates are parsed in ISO format
- Output files are created in output folder

## Troubleshooting

Common issues and solutions documented in README.md:
- Dependency installation
- File format support
- Encoding issues
- Date parsing
- Category detection
- Docker networking

---

**Service Status**: Ready for deployment ✅
**Documentation**: Complete ✅
**Tests**: Included ✅
**Examples**: Provided ✅
**Docker Integration**: Configured ✅
