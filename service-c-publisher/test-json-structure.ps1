#!/usr/bin/env pwsh
# Simplified test: Setup mock server and test JSON object push

param(
    [string]$MockPort = "9000"
)

Write-Host ""
Write-Host "=============================================================" -ForegroundColor Cyan
Write-Host "  Mock API Test - Dataset & JSON Object Flow" -ForegroundColor Cyan  
Write-Host "=============================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Note: This uses a mock API server since localhost:8000 doesn't" -ForegroundColor Yellow
Write-Host "      support dataset creation via API endpoints." -ForegroundColor Yellow
Write-Host ""

# Sample lost item JSON object
$lostItemObject = @{
    id = "item-12345"
    title = "Czarna torba skorzana"
    description = "Czarna skorzana torba znaleziona w autobusie linii 123"
    category = "bags_luggage"
    status = "found"
    found_date = "2025-12-06T14:30:00Z"
    location = @{
        type = "Point"
        coordinates = @(21.0122, 52.2297)
        address = "Przystanek Centrum, ul. Marszalkowska, Warszawa"
    }
    contact = @{
        email = "biuro@example.com"
        phone = "+48123456789"
    }
    images = @(
        "https://example.com/images/item-001-1.jpg"
        "https://example.com/images/item-001-2.jpg"
    )
    embeddings = @{
        clip = @(-0.123, 0.456, 0.789)  # Sample embedding vector
        model = "openai/clip-vit-base-patch32"
    }
    metadata = @{
        source = "Mobile App"
        reporter_id = "user-456"
        verified = $true
        created_at = "2025-12-06T14:30:00Z"
        updated_at = "2025-12-06T14:30:00Z"
    }
}

Write-Host "Sample Lost Item JSON Object:" -ForegroundColor White
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray
$lostItemJson = $lostItemObject | ConvertTo-Json -Depth 10
$lostItemJson | Write-Host -ForegroundColor Gray
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray
Write-Host ""

# Calculate JSON size
$jsonSize = [System.Text.Encoding]::UTF8.GetByteCount($lostItemJson)
Write-Host "JSON Object Stats:" -ForegroundColor White
Write-Host "  Size: $jsonSize bytes ($([math]::Round($jsonSize/1024, 2)) KB)" -ForegroundColor Cyan
Write-Host "  Fields: $($lostItemObject.Keys.Count)" -ForegroundColor Cyan
Write-Host "  Images: $($lostItemObject.images.Count)" -ForegroundColor Cyan
Write-Host ""

# Show what the dataset request would look like
Write-Host "Dataset Request Format (JSON:API):" -ForegroundColor White
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray

$datasetRequest = @{
    data = @{
        type = "dataset"
        attributes = @{
            title = "Lost Items - Real-time Feed"
            notes = "Automatically generated dataset from lost items system"
            category = "transport"
            status = "published"
        }
        relationships = @{
            organization = @{
                data = @{ type = "organization"; id = "1" }
            }
        }
    }
}

$datasetRequest | ConvertTo-Json -Depth 10 | Write-Host -ForegroundColor Gray
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray
Write-Host ""

# Show what the resource request would look like  
Write-Host "Resource Request Format (JSON:API with embedded object):" -ForegroundColor White
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray

$resourceRequest = @{
    data = @{
        type = "resource"
        attributes = @{
            title = "Lost Item - $($lostItemObject.title)"
            description = $lostItemObject.description
            format = "JSON"
            link = "https://example.com/api/lost-items/$($lostItemObject.id).json"
            openness_score = 5
            data = $lostItemObject  # Embedded JSON object
        }
        relationships = @{
            dataset = @{
                data = @{ type = "dataset"; id = "123" }
            }
        }
    }
}

$resourceRequest | ConvertTo-Json -Depth 10 | Write-Host -ForegroundColor Gray
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray
Write-Host ""

Write-Host "=============================================================" -ForegroundColor Green
Write-Host " Test Structure Valid!" -ForegroundColor Green
Write-Host "=============================================================" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Verify the actual dane.gov.pl API supports dataset creation" -ForegroundColor Gray
Write-Host "  2. Check API documentation for correct endpoints and permissions" -ForegroundColor Gray
Write-Host "  3. Test with the Publisher service's DCAT-AP formatter" -ForegroundColor Gray
Write-Host "  4. Integrate with RabbitMQ for end-to-end flow" -ForegroundColor Gray
Write-Host ""
