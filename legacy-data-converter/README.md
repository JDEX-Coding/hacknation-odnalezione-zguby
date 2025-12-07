# Legacy Data Converter

Python service for extracting text from various file formats and converting them to the lost-items schema using NLP.

**⚠️ IMPORTANT:** This service operates as a **RabbitMQ consumer**, not a REST API. It processes files from the `q.datasets.process` queue.

## Features

- **Text Extraction** from multiple formats:
  - PDF documents
  - DOCX (Word) files
  - HTML files
  - Plain text (TXT)
  - JSON documents
  - XML files
  - CSV spreadsheets

- **NLP-based Conversion** to lost-items schema:
  - Automatic field extraction (title, description, category, location, dates, contact info)
  - Category classification using keyword matching
  - Date parsing with multiple format support
  - Contact information extraction (email, phone)
  - Polish language support

- **RabbitMQ Integration**:
  - Consumes from `q.datasets.process` queue
  - Publishes converted items to `item.submitted` routing key
  - Each item flows through CLIP → Qdrant → Publisher pipeline
  - Handles multiple items per file (e.g., CSV with multiple rows)

## Architecture

```
Gateway/User → [dataset.submitted] → q.datasets.process → Legacy Converter
                                                                    ↓
                                                      [Extracts & Converts]
                                                                    ↓
                                            [item.submitted] → q.lost-items.embed
                                                                    ↓
                                                              CLIP Service
                                                                    ↓
                                            [item.embedded] → q.lost-items.ingest
                                                                    ↓
                                                            Qdrant Service
                                                                    ↓
                                        [item.vectorized] → q.lost-items.publish
                                                                    ↓
                                                            Publisher Service
```

## Installation

### Using Docker (Recommended)

Build and run with docker-compose:

```bash
docker-compose up legacy-converter
```

The service runs as a background consumer (no HTTP port exposed).

### Local Installation

#### Basic installation (without heavy NLP dependencies):

```bash
cd legacy-data-converter
pip install -r requirements-basic.txt
python main.py
```

#### Full installation (with spaCy and transformers):

```bash
cd legacy-data-converter
pip install -r requirements.txt
python -m spacy download pl_core_news_sm
python main.py
```

## Usage

### Publish Dataset to RabbitMQ

#### Using Test Script

```bash
# Install pika
pip install pika

# Publish a file
python test_publish.py examples/lost_items.csv test-001
```

#### Message Format

Publish to exchange `lost-found.events` with routing key `dataset.submitted`:

```json
{
  "dataset_id": "test-001",
  "file_data": "base64-encoded-file-content",
  "file_name": "lost_items.csv",
  "file_format": ".csv"
}
```

#### Example with Python

```python
import pika
import json
import base64

# Read and encode file
with open('examples/lost_items.csv', 'rb') as f:
    file_data = base64.b64encode(f.read()).decode('utf-8')

# Create message
message = {
    'dataset_id': 'test-001',
    'file_data': file_data,
    'file_name': 'lost_items.csv',
    'file_format': '.csv'
}

# Publish to RabbitMQ
connection = pika.BlockingConnection(
    pika.URLParameters('amqp://admin:admin123@localhost:5674/')
)
channel = connection.channel()
channel.basic_publish(
    exchange='lost-found.events',
    routing_key='dataset.submitted',
    body=json.dumps(message)
)
connection.close()
```

### Command-Line Usage

#### Extract text from a file:
```bash
python text_extractor.py path/to/file.pdf
```

#### Convert file to lost-items schema:
```bash
python nlp_converter.py path/to/file.pdf
```

## Configuration

Environment variables:

- `PORT` - Service port (default: 8083)
- `UPLOAD_FOLDER` - Temporary upload folder (default: ./uploads)
- `OUTPUT_FOLDER` - Output folder for converted files (default: ./output)
- `MAX_FILE_SIZE` - Maximum file size in bytes (default: 10485760 = 10MB)
- `DEBUG` - Enable debug mode (default: false)

## Supported Categories

The converter recognizes the following categories:

- **dokumenty** - Documents (IDs, passports, licenses)
- **elektronika** - Electronics (phones, laptops, tablets)
- **biżuteria** - Jewelry (rings, watches, necklaces)
- **odzież** - Clothing (jackets, shoes, accessories)
- **torby i plecaki** - Bags and backpacks
- **klucze** - Keys
- **portfele** - Wallets
- **telefony** - Phones
- **zwierzęta** - Animals/Pets
- **pojazdy** - Vehicles (bikes, scooters)
- **inne** - Other

## Output Schema

The converter produces items in the following format:

```json
{
  "id": "uuid",
  "title": "Item title",
  "description": "Item description",
  "category": "elektronika",
  "location": "ul. Marszałkowska 1, Warszawa",
  "found_date": "2024-01-15T00:00:00",
  "reporting_date": "2024-01-16T00:00:00",
  "reporting_location": "Komisariat Policji",
  "contact_email": "kontakt@example.com",
  "contact_phone": "+48123456789",
  "status": "pending",
  "created_at": "2024-01-16T10:30:00",
  "updated_at": "2024-01-16T10:30:00",
  "metadata": {
    "source_file": "document.pdf",
    "source_format": ".pdf",
    "conversion_timestamp": "2024-01-16T10:30:00"
  }
}
```

## Architecture

The service consists of three main components:

1. **text_extractor.py** - Handles text extraction from various file formats
2. **nlp_converter.py** - Converts extracted text to lost-items schema using NLP
3. **main.py** - Flask API server providing REST endpoints

### Text Extraction

The `TextExtractor` class supports multiple file formats:

- **PDF**: Uses PyPDF2 to extract text from all pages
- **DOCX**: Uses python-docx to extract paragraphs and tables
- **HTML**: Uses BeautifulSoup to extract clean text
- **TXT**: Handles multiple encodings (UTF-8, Latin-1, CP1252)
- **JSON**: Parses and converts to readable text
- **XML**: Extracts text content and converts to structured format
- **CSV**: Reads and converts to text and structured data

### NLP Conversion

The `NLPConverter` class uses multiple techniques:

1. **Structured Data Mapping**: For CSV/JSON/XML, attempts to map fields directly
2. **Pattern Matching**: Uses regex patterns for dates, locations, contacts
3. **Keyword Classification**: Matches categories based on keywords
4. **Optional spaCy**: For advanced entity recognition (Polish language)
5. **Optional Transformers**: For zero-shot classification

### Workflow

```
File Upload → Text Extraction → NLP Conversion → Lost Item Schema
     ↓              ↓                   ↓                ↓
  Save File    Extract Text      Identify Fields    Return JSON
     ↓              ↓                   ↓                ↓
  Process       Parse Format      Map to Schema    Save Output
     ↓              ↓                   ↓                ↓
  Cleanup      Return Data        Validate         Cleanup
```

## Development

### Run tests:
```bash
make test
```

### Format code:
```bash
black *.py
```

### Lint code:
```bash
flake8 *.py
```

## Integration with Other Services

The legacy-converter service can be integrated with the gateway service to automatically convert legacy data files when uploaded. The converted items can then be:

1. Stored in PostgreSQL database
2. Processed by CLIP service for image embedding
3. Vectorized by Qdrant service
4. Published to dane.gov.pl via the publisher service

## Examples

### Example 1: Converting a CSV file with lost items

CSV content:
```csv
Przedmiot,Opis,Kategoria,Miejsce,Data znalezienia,Email
Portfel skórzany,Czarny portfel z dokumentami,portfele,ul. Marszałkowska 1,2024-01-15,biuro@urzad.pl
Telefon Samsung,Galaxy S21 w etui,telefony,Dworzec Centralny,2024-01-14,kontakt@urzad.pl
```

Result:
```json
[
  {
    "title": "Portfel skórzany",
    "description": "Czarny portfel z dokumentami",
    "category": "portfele",
    "location": "ul. Marszałkowska 1",
    "found_date": "2024-01-15T00:00:00",
    "contact_email": "biuro@urzad.pl"
  },
  {
    "title": "Telefon Samsung",
    "description": "Galaxy S21 w etui",
    "category": "telefony",
    "location": "Dworzec Centralny",
    "found_date": "2024-01-14T00:00:00",
    "contact_email": "kontakt@urzad.pl"
  }
]
```

### Example 2: Converting unstructured text

Input text:
```
Znaleziono dowód osobisty Jana Kowalskiego.
Dokument został znaleziony 15 stycznia 2024 roku przy ulicy Nowy Świat 10 w Warszawie.
Kontakt: biuro@urzad.warszawa.pl
```

Result:
```json
{
  "title": "Znaleziono dowód osobisty Jana Kowalskiego.",
  "description": "Znaleziono dowód osobisty Jana Kowalskiego. Dokument został znaleziony 15 stycznia 2024 roku przy ulicy Nowy Świat 10 w Warszawie. Kontakt: biuro@urzad.warszawa.pl",
  "category": "dokumenty",
  "location": "ulicy Nowy Świat 10",
  "found_date": "2024-01-15T00:00:00",
  "contact_email": "biuro@urzad.warszawa.pl"
}
```

## Troubleshooting

### Issue: "PyPDF2 not installed"
Solution: Install with `pip install PyPDF2` or use requirements.txt

### Issue: "spaCy model not found"
Solution: Download Polish model with `python -m spacy download pl_core_news_sm`

### Issue: File upload fails
- Check MAX_FILE_SIZE setting
- Ensure file extension is supported
- Verify file is not corrupted

### Issue: Poor category detection
- Use full NLP installation with spaCy/transformers
- Add custom category keywords
- Provide structured data with explicit category field

## License

See LICENSE file in the root directory.

## Contributing

Contributions are welcome! Please ensure:
- Code follows PEP 8 style guide
- Tests pass
- Documentation is updated
