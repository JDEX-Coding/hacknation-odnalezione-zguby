#!/usr/bin/env pwsh
# Complete test script: Login -> Create Dataset -> Push JSON Object as Resource

param(
    [string]$ApiUrl = "http://localhost:8000",
    [string]$Email = "admin@mcod.local",
    [string]$Password = "Hacknation-2025"
)

Write-Host ""
Write-Host "=============================================================" -ForegroundColor Cyan
Write-Host "  Dataset Setup & JSON Object Push Test" -ForegroundColor Cyan
Write-Host "=============================================================" -ForegroundColor Cyan
Write-Host ""

# ============================================================================
# STEP 1: LOGIN
# ============================================================================
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray
Write-Host " STEP 1: Login & Get Bearer Token" -ForegroundColor Yellow
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray

$loginPayload = @{
    data = @{
        type = "login"
        attributes = @{
            email = $Email
            password = $Password
        }
    }
} | ConvertTo-Json -Depth 10

Write-Host "Authenticating as: $Email" -ForegroundColor Gray

try {
    $loginResponse = Invoke-RestMethod -Uri "$ApiUrl/auth/login" `
        -Method POST `
        -ContentType "application/json" `
        -Body $loginPayload

    $token = $loginResponse.data.attributes.token
    Write-Host "[OK] Login successful!" -ForegroundColor Green
    Write-Host "Token: $($token.Substring(0, 50))..." -ForegroundColor DarkGray
    Write-Host ""
}
catch {
    Write-Host "[FAIL] Login failed: $_" -ForegroundColor Red
    exit 1
}

# ============================================================================
# STEP 2: CREATE DATASET
# ============================================================================
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray
Write-Host " STEP 2: Create Dataset" -ForegroundColor Yellow
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray

$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$datasetPayload = @{
    data = @{
        type = "dataset"
        attributes = @{
            title = "Lost Items API - Test Dataset $timestamp"
            notes = "Test dataset for pushing JSON objects with lost item data"
            url = "https://example.com/lost-items-api"
            category = "transport"
            update_frequency = "daily"
            license_condition_db_or_copyrighted = $false
            license_condition_modification = $true
            license_condition_responsibilities = $false
            license_condition_personal_data = $false
            license_condition_source = $true
            status = "published"
            tags = @(
                @{ name = "test"; display_name = "test" }
                @{ name = "lost items"; display_name = "lost items" }
            )
        }
        relationships = @{
            organization = @{
                data = @{
                    type = "organization"
                    id = "1"
                }
            }
        }
    }
} | ConvertTo-Json -Depth 10

Write-Host "Creating dataset..." -ForegroundColor Gray

try {
    $datasetResponse = Invoke-RestMethod -Uri "$ApiUrl/api/1.4/datasets" `
        -Method POST `
        -Headers @{ 
            "Authorization" = "Bearer $token"
            "Content-Type" = "application/json"
        } `
        -Body $datasetPayload

    $datasetId = $datasetResponse.data.id
    Write-Host "[OK] Dataset created!" -ForegroundColor Green
    Write-Host "Dataset ID: $datasetId" -ForegroundColor Cyan
    Write-Host "Title: $($datasetResponse.data.attributes.title)" -ForegroundColor DarkGray
    Write-Host ""
}
catch {
    Write-Host "[FAIL] Dataset creation failed: $_" -ForegroundColor Red
    if ($_.ErrorDetails.Message) {
        $_.ErrorDetails.Message | Write-Host -ForegroundColor DarkRed
    }
    exit 1
}

# ============================================================================
# STEP 3: CREATE JSON OBJECT AS RESOURCE
# ============================================================================
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray
Write-Host " STEP 3: Push JSON Object to Dataset" -ForegroundColor Yellow
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray

# Sample lost item object
$lostItemObject = @{
    id = "item-001"
    title = "Czarna torba skorzana"
    description = "Czarna skorzana torba znaleziona w autobusie linii 123. Zawiera dokumenty i telefon."
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
    metadata = @{
        source = "Mobile App"
        reporter_id = "user-456"
        verified = $true
    }
}

Write-Host "Sample JSON Object:" -ForegroundColor Gray
$lostItemObject | ConvertTo-Json -Depth 10 | Write-Host -ForegroundColor DarkGray
Write-Host ""

# Create resource with JSON data
$resourcePayload = @{
    data = @{
        type = "resource"
        attributes = @{
            title = "Lost Item Data - JSON Object"
            description = "JSON object containing structured data about a lost item"
            format = "JSON"
            link = "https://example.com/api/lost-items/item-001.json"
            openness_score = 5
            data = $lostItemObject  # Embed the actual JSON object
        }
        relationships = @{
            dataset = @{
                data = @{
                    type = "dataset"
                    id = $datasetId
                }
            }
        }
    }
} | ConvertTo-Json -Depth 10

Write-Host "Pushing JSON object as resource..." -ForegroundColor Gray

try {
    $resourceResponse = Invoke-RestMethod -Uri "$ApiUrl/api/1.4/resources" `
        -Method POST `
        -Headers @{ 
            "Authorization" = "Bearer $token"
            "Content-Type" = "application/json"
        } `
        -Body $resourcePayload

    $resourceId = $resourceResponse.data.id
    Write-Host "[OK] JSON object pushed successfully!" -ForegroundColor Green
    Write-Host "Resource ID: $resourceId" -ForegroundColor Cyan
    Write-Host "Format: $($resourceResponse.data.attributes.format)" -ForegroundColor DarkGray
    Write-Host ""
}
catch {
    Write-Host "[FAIL] Resource creation failed: $_" -ForegroundColor Red
    if ($_.ErrorDetails.Message) {
        $_.ErrorDetails.Message | Write-Host -ForegroundColor DarkRed
    }
    exit 1
}

# ============================================================================
# STEP 4: VERIFY - GET DATASET WITH RESOURCES
# ============================================================================
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray
Write-Host " STEP 4: Verify Dataset & Resources" -ForegroundColor Yellow
Write-Host "-------------------------------------------------------------" -ForegroundColor DarkGray

try {
    $verifyResponse = Invoke-RestMethod -Uri "$ApiUrl/api/1.4/datasets/$datasetId" `
        -Method GET `
        -Headers @{ 
            "Authorization" = "Bearer $token"
            "Accept" = "application/json"
        }

    Write-Host "[OK] Dataset verification successful!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Dataset Details:" -ForegroundColor White
    Write-Host "  ID: $($verifyResponse.data.id)" -ForegroundColor Gray
    Write-Host "  Title: $($verifyResponse.data.attributes.title)" -ForegroundColor Gray
    Write-Host "  Status: $($verifyResponse.data.attributes.status)" -ForegroundColor Gray
    
    if ($verifyResponse.data.relationships.resources) {
        Write-Host "  Resources Count: $($verifyResponse.data.relationships.resources.meta.count)" -ForegroundColor Gray
    }
    Write-Host ""
}
catch {
    Write-Host "[WARN] Could not verify dataset: $_" -ForegroundColor Yellow
    Write-Host ""
}

# ============================================================================
# SUMMARY
# ============================================================================
Write-Host "=============================================================" -ForegroundColor Green
Write-Host " SUCCESS! Complete Flow Test Passed" -ForegroundColor Green
Write-Host "=============================================================" -ForegroundColor Green
Write-Host ""
Write-Host "Summary:" -ForegroundColor White
Write-Host "  Dataset ID:  $datasetId" -ForegroundColor Cyan
Write-Host "  Resource ID: $resourceId" -ForegroundColor Cyan
Write-Host "  Format:      JSON" -ForegroundColor Cyan
Write-Host ""
Write-Host "  View at: $ApiUrl/dataset/$datasetId" -ForegroundColor Gray
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Check the API to verify the dataset was created" -ForegroundColor Gray
Write-Host "  2. Verify the JSON object is accessible via the resource" -ForegroundColor Gray
Write-Host "  3. Test with real RabbitMQ events for full integration" -ForegroundColor Gray
Write-Host ""
