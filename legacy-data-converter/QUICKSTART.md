# Legacy Data Converter - Quick Start Guide

## Quick Start

### 1. Start the Service

Using docker-compose:
```powershell
docker-compose up legacy-converter
```

Or build and run separately:
```powershell
cd legacy-data-converter
docker build -t legacy-converter .
docker run -p 8083:8083 legacy-converter
```

### 2. Test the Service

Check if service is running:
```powershell
curl http://localhost:8083/health
```

### 3. Convert a File

#### Example: Convert a text file

Create a test file `test_item.txt`:
```
Znaleziono portfel męski w kolorze czarnym.
Portfel został znaleziony w dniu 15 stycznia 2024 roku
przy ulicy Marszałkowskiej 10 w Warszawie.

W środku znajdują się dokumenty i karty.

Kontakt: biuro@urzad.warszawa.pl
Telefon: 22-123-45-67
```

Convert it:
```powershell
curl -X POST http://localhost:8083/convert `
  -F "file=@test_item.txt"
```

#### Example: Convert CSV with multiple items

Create `lost_items.csv`:
```csv
Przedmiot,Opis,Kategoria,Miejsce,Data znalezienia,Email
Portfel skórzany,Czarny portfel z dokumentami,portfele,ul. Marszałkowska 1,2024-01-15,biuro@urzad.pl
Telefon Samsung,Galaxy S21 w niebieskim etui,telefony,Dworzec Centralny,2024-01-14,kontakt@urzad.pl
Klucze,Pęk kluczy z breloczkiem,klucze,Park Łazienkowski,2024-01-13,zguby@urzad.pl
```

Convert multiple files:
```powershell
curl -X POST http://localhost:8083/convert/batch `
  -F "files=@lost_items.csv" `
  -F "files=@test_item.txt" `
  -F "dataset_id=batch-2024-01"
```

### 4. Integration with Gateway Service

The legacy converter can work with the main gateway to process legacy data:

```powershell
# 1. Convert legacy files to JSON
curl -X POST http://localhost:8083/convert/batch `
  -F "files=@legacy_data.csv" `
  -F "dataset_id=migration-2024" `
  -o converted_items.json

# 2. Submit items to gateway (manual integration)
# Parse converted_items.json and submit each item via gateway API
```

### 5. View Output

Converted files are saved to the output folder:
```powershell
docker exec odnalezione-legacy-converter ls /app/output
```

Copy output from container:
```powershell
docker cp odnalezione-legacy-converter:/app/output/. ./output/
```

## Testing Examples

### Test with different file formats

#### PDF Example
```powershell
curl -X POST http://localhost:8083/convert `
  -F "file=@document.pdf"
```

#### DOCX Example
```powershell
curl -X POST http://localhost:8083/convert `
  -F "file=@document.docx"
```

#### JSON Example
```powershell
curl -X POST http://localhost:8083/convert `
  -F "file=@items.json"
```

#### XML Example
```powershell
curl -X POST http://localhost:8083/convert `
  -F "file=@data.xml"
```

### Test text extraction only

```powershell
curl -X POST http://localhost:8083/extract `
  -F "file=@document.pdf"
```

### Get supported categories

```powershell
curl http://localhost:8083/categories
```

## Common Use Cases

### Use Case 1: Migrate Legacy CSV Data

1. Export existing database to CSV
2. Convert CSV to lost-items schema:
```powershell
curl -X POST http://localhost:8083/convert `
  -F "file=@legacy_database.csv" `
  -F "dataset_id=legacy-migration" `
  -o converted.json
```
3. Import converted items to system

### Use Case 2: Process Scanned Documents

1. OCR scan to PDF/DOCX (external tool)
2. Extract and convert:
```powershell
curl -X POST http://localhost:8083/convert/batch `
  -F "files=@scan1.pdf" `
  -F "files=@scan2.pdf" `
  -F "files=@scan3.pdf"
```

### Use Case 3: Import from Excel/CSV Reports

1. Save Excel as CSV
2. Convert with category detection:
```powershell
curl -X POST http://localhost:8083/convert `
  -F "file=@monthly_report.csv" `
  -F "dataset_id=report-2024-01"
```

## Troubleshooting

### Service won't start
```powershell
# Check logs
docker logs odnalezione-legacy-converter

# Rebuild container
docker-compose build legacy-converter
docker-compose up legacy-converter
```

### File upload fails
```powershell
# Check file size (default limit: 10MB)
# Increase MAX_FILE_SIZE in docker-compose.yml

# Check file format is supported
curl http://localhost:8083/categories
```

### Poor conversion quality
- Use structured data (CSV, JSON, XML) when possible
- Ensure dates are in standard formats (DD.MM.YYYY or YYYY-MM-DD)
- Include clear field labels in CSV headers
- For text files, use clear formatting with labels

## Next Steps

1. **Integration**: Connect with gateway service for automatic processing
2. **Automation**: Create scripts to batch-process directories of files
3. **Validation**: Review converted items before submitting to database
4. **Customization**: Add custom category keywords or parsing rules

## API Response Examples

### Successful Conversion
```json
{
  "success": true,
  "item": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "title": "Portfel skórzany",
    "description": "Czarny portfel z dokumentami",
    "category": "portfele",
    "location": "ul. Marszałkowska 1",
    "found_date": "2024-01-15T00:00:00",
    "contact_email": "biuro@urzad.pl",
    "status": "pending",
    "created_at": "2024-01-16T10:30:00",
    "updated_at": "2024-01-16T10:30:00",
    "metadata": {
      "source_file": "lost_items.csv",
      "source_format": ".csv",
      "conversion_timestamp": "2024-01-16T10:30:00"
    }
  },
  "output_file": "abc123_converted.json"
}
```

### Batch Conversion
```json
{
  "success": true,
  "items_converted": 3,
  "items": [
    { "id": "...", "title": "..." },
    { "id": "...", "title": "..." },
    { "id": "...", "title": "..." }
  ],
  "errors": null,
  "output_file": "batch_20240116_103000.json"
}
```

## Performance Tips

- Use batch endpoint for multiple files
- Pre-process files to remove unnecessary content
- Use CSV/JSON for best structured data conversion
- Monitor output folder size and clean periodically
- Consider increasing MAX_FILE_SIZE for large documents
