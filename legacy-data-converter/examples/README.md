# Legacy Data Converter - Examples

This folder contains example files for testing the legacy data converter service.

## Files

### `lost_items.csv`
Example CSV file with multiple lost items. Demonstrates:
- Structured data with clear column headers
- Multiple items in one file
- Various categories
- Polish language content

### `items.json`
Example JSON file with lost items. Demonstrates:
- JSON array format
- Structured data
- Optional fields (some items missing phone)

### `items.xml`
Example XML file with lost items. Demonstrates:
- Nested XML structure
- Multiple items
- Contact information grouping

### `report.txt`
Example unstructured text file. Demonstrates:
- Free-form text
- Date in natural language (Polish)
- Address extraction
- Contact information extraction

## Testing

### Test single file conversion:
```powershell
# CSV
curl -X POST http://localhost:8083/convert -F "file=@lost_items.csv"

# JSON
curl -X POST http://localhost:8083/convert -F "file=@items.json"

# XML
curl -X POST http://localhost:8083/convert -F "file=@items.xml"

# TXT
curl -X POST http://localhost:8083/convert -F "file=@report.txt"
```

### Test batch conversion:
```powershell
curl -X POST http://localhost:8083/convert/batch `
  -F "files=@lost_items.csv" `
  -F "files=@items.json" `
  -F "files=@items.xml" `
  -F "files=@report.txt" `
  -F "dataset_id=test-examples"
```

### Test text extraction only:
```powershell
curl -X POST http://localhost:8083/extract -F "file=@report.txt"
```

## Expected Results

### CSV Conversion
Should extract 8 items with properly mapped fields from the CSV structure.

### JSON Conversion
Should extract 3 items with direct field mapping.

### XML Conversion
Should extract 3 items, parsing the nested XML structure.

### TXT Conversion
Should extract 1 item using NLP techniques:
- Title from first significant line
- Description from full text
- Category: "portfele" (detected from keywords)
- Location: "ul. Marszałkowska 10, Warszawa"
- Date: "2024-01-15"
- Email: "biuro@urzad.warszawa.pl"
- Phone: "22-123-45-67"

## Creating Your Own Test Files

### CSV Format
```csv
Przedmiot,Opis,Kategoria,Miejsce,Data znalezienia,Email,Telefon
[Item name],[Description],[Category],[Location],[Date],[Email],[Phone]
```

### JSON Format
```json
[
  {
    "title": "Item name",
    "description": "Description",
    "category": "category",
    "location": "Location",
    "found_date": "YYYY-MM-DD",
    "contact_email": "email@example.com",
    "contact_phone": "123456789"
  }
]
```

### XML Format
```xml
<?xml version="1.0" encoding="UTF-8"?>
<lost_items>
  <item>
    <title>Item name</title>
    <description>Description</description>
    <category>category</category>
    <location>Location</location>
    <found_date>YYYY-MM-DD</found_date>
    <contact>
      <email>email@example.com</email>
      <phone>123456789</phone>
    </contact>
  </item>
</lost_items>
```

### TXT Format
Free-form text with relevant information. The NLP converter will attempt to extract:
- Item description
- Location (look for "przy", "na", "w" + address)
- Date (various formats supported)
- Contact information (email, phone patterns)

## Challenging Examples

The `messy_*` files contain intentionally difficult, real-world examples:

### `messy_report1.txt`
- Police patrol report format
- Informal language mixed with formal
- Multiple contact formats
- Date without year (15.01.24)
- Location described narratively

### `messy_report2.txt`
- Service note style
- Very informal language ("!!!", "PS.")
- Colloquial date format ("dwunastego stycznia br.")
- Missing structured fields
- Contact info buried in text

### `messy_report3.txt`
- Multi-item report (2 items in one document)
- Complex formatting with ASCII art
- Uncertain information ("mogą należeć... lub nie???")
- Recommendations mixed with data
- Multiple contact methods

### `messy_email.html`
- HTML email format
- Embedded styling
- Bullet points in HTML
- Yellow highlighting
- Auto-generated footer
- Informal communication style

### `messy_spreadsheet.csv`
- Inconsistent date formats (13-01-2024, 14.01, 15/01/2024, 2024-01-16, etc.)
- Missing data (empty cells)
- Messy notes and comments
- Mixed phone number formats
- Status notes in description field
- Variable column usage

### `messy_legacy_export.xml`
- Old system export format
- Empty title fields
- Long narrative descriptions
- Inconsistent field naming
- Mixed date formats
- Status fields in Polish
- Nested contact information

### `messy_legacy_system.json`
- Legacy JSON with non-standard structure
- Variable field names (ref_number formats)
- Missing reference numbers
- Extensive notes and comments
- Multiple item descriptions in single field
- Priority markers
- Summary metadata

## Testing Messy Files

Test the converter's ability to handle imperfect data:

```bash
# Test challenging reports
python test_publish.py examples/messy_report1.txt messy-test-1
python test_publish.py examples/messy_report2.txt messy-test-2
python test_publish.py examples/messy_report3.txt messy-test-3

# Test messy HTML email
python test_publish.py examples/messy_email.html messy-test-4

# Test inconsistent spreadsheet
python test_publish.py examples/messy_spreadsheet.csv messy-test-5

# Test legacy exports
python test_publish.py examples/messy_legacy_export.xml messy-test-6
python test_publish.py examples/messy_legacy_system.json messy-test-7
```

These examples test the converter's robustness in handling:
- ✅ Multiple date formats
- ✅ Inconsistent field naming
- ✅ Missing required fields
- ✅ Informal language
- ✅ Multi-item documents
- ✅ HTML formatting
- ✅ Embedded notes and comments
- ✅ Mixed data quality
