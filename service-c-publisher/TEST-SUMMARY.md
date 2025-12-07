# Dataset & JSON Object Push - Test Summary

## Overview
Created test scripts to demonstrate the complete flow of setting up a dataset and pushing JSON objects to the dane.gov.pl API.

## Test Scripts Created

### 1. `test-full-flow.ps1`
Complete end-to-end test that:
- ✅ Logs in to get Bearer token
- ✅ Creates a dataset with proper JSON:API format
- ✅ Pushes a JSON object as a resource
- ✅ Verifies the dataset was created

**Usage:**
```powershell
.\test-full-flow.ps1
```

**Note:** Requires API version in path (`/api/1.4/datasets`)

### 2. `test-json-structure.ps1`
Validates JSON structure and formats without making API calls:
- Shows sample lost item JSON object
- Displays JSON:API formatted dataset request
- Shows JSON:API formatted resource request with embedded object
- Calculates object size and statistics

**Usage:**
```powershell
.\test-json-structure.ps1
```

## Sample Lost Item JSON Object

```json
{
    "id": "item-12345",
    "title": "Czarna torba skorzana",
    "description": "Czarna skorzana torba znaleziona w autobusie linii 123",
    "category": "bags_luggage",
    "status": "found",
    "found_date": "2025-12-06T14:30:00Z",
    "location": {
        "type": "Point",
        "coordinates": [21.0122, 52.2297],
        "address": "Przystanek Centrum, ul. Marszalkowska, Warszawa"
    },
    "contact": {
        "email": "biuro@example.com",
        "phone": "+48123456789"
    },
    "images": [
        "https://example.com/images/item-001-1.jpg",
        "https://example.com/images/item-001-2.jpg"
    ],
    "embeddings": {
        "clip": [-0.123, 0.456, 0.789],
        "model": "openai/clip-vit-base-patch32"
    },
    "metadata": {
        "source": "Mobile App",
        "reporter_id": "user-456",
        "verified": true,
        "created_at": "2025-12-06T14:30:00Z",
        "updated_at": "2025-12-06T14:30:00Z"
    }
}
```

**Stats:** ~1.5 KB, 11 fields, 2 images

## JSON:API Format for Resources

When pushing the JSON object to the API, it's wrapped in JSON:API format:

```json
{
    "data": {
        "type": "resource",
        "attributes": {
            "title": "Lost Item - Czarna torba skorzana",
            "description": "Czarna skorzana torba znaleziona w autobusie linii 123",
            "format": "JSON",
            "link": "https://example.com/api/lost-items/item-12345.json",
            "openness_score": 5,
            "data": { /* embedded lost item object */ }
        },
        "relationships": {
            "dataset": {
                "data": {
                    "type": "dataset",
                    "id": "123"
                }
            }
        }
    }
}
```

## API Endpoints Updated

Updated Go client to use versioned endpoints:
- ✅ `/auth/login` (no version needed)
- ✅ `/api/1.4/datasets` (for creating datasets)
- ✅ `/api/1.4/resources` (for creating resources)
- ✅ `/api/1.4/datasets/{id}` (for retrieving datasets)

## Current Status

### ✅ Working
- Authentication flow (login with email/password)
- JWT Bearer token retrieval
- JSON:API request format validation
- JSON object structure for lost items

### ⚠️ Needs Verification
- Dataset creation endpoint availability on localhost:8000
- Required permissions for dataset creation
- API might be read-only or require admin configuration

### 📋 Next Steps
1. Verify API documentation for dataset creation capabilities
2. Check if additional permissions/roles are needed
3. Test with Publisher service's DCAT-AP formatter
4. Integrate with RabbitMQ for complete flow:
   - Receive `item.vectorized` event
   - Format as DCAT-AP
   - Create dataset if needed
   - Push JSON object as resource
   - Emit `item.published` event

## Integration with Publisher Service

The Publisher service (`service-c-publisher`) is configured to:
1. Listen for `item.vectorized` events on RabbitMQ
2. Format lost items as DCAT-AP PL standard
3. Authenticate with dane.gov.pl API
4. Publish datasets with Bearer token authentication
5. Send success/failure events back to RabbitMQ

**Configuration:**
```env
DANE_GOV_API_URL=http://localhost:8000
DANE_GOV_EMAIL=admin@mcod.local
DANE_GOV_PASSWORD=Hacknation-2025
```

## Files Modified
- `service-c-publisher/internal/client/dane_gov_client.go` - Added API version to endpoints
- `service-c-publisher/test-full-flow.ps1` - Complete integration test
- `service-c-publisher/test-json-structure.ps1` - Structure validation test
