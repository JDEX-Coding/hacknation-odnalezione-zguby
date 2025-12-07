#!/usr/bin/env pwsh
# Test script for dataset publication pipeline
# Gateway -> RabbitMQ -> Publisher -> dane.gov.pl

param(
    [string]$GatewayUrl = "http://localhost:8082",
    [string]$DatasetID = ""
)

Write-Host "`n🧪 Dataset Publication Pipeline Test" -ForegroundColor Cyan
Write-Host "=" * 70 -ForegroundColor DarkGray

# Step 1: List all datasets
Write-Host "`n📋 Step 1: Listing all datasets..." -ForegroundColor Yellow

try {
    $datasetsResponse = Invoke-RestMethod -Uri "$GatewayUrl/api/datasets" `
        -Method GET `
        -ContentType "application/json"
    
    Write-Host "✅ Found $($datasetsResponse.count) dataset(s)" -ForegroundColor Green
    
    if ($datasetsResponse.count -eq 0) {
        Write-Host "❌ No datasets found. Please create a dataset first." -ForegroundColor Red
        Write-Host "   Use: /create-dataset endpoint" -ForegroundColor Gray
        exit 1
    }
    
    # Display datasets
    Write-Host "`nAvailable datasets:" -ForegroundColor White
    foreach ($ds in $datasetsResponse.datasets) {
        Write-Host "  ID: $($ds.id)" -ForegroundColor Cyan
        Write-Host "  Title: $($ds.title)" -ForegroundColor White
        Write-Host "  Institution: $($ds.institution_name)" -ForegroundColor Gray
        Write-Host "  Email: $($ds.email)" -ForegroundColor Gray
        Write-Host "  Categories: $($ds.categories -join ', ')" -ForegroundColor Gray
        Write-Host "  Tags: $($ds.tags -join ', ')" -ForegroundColor Gray
        Write-Host "  ---" -ForegroundColor DarkGray
    }
    
    # Use first dataset if not specified
    if ($DatasetID -eq "") {
        $DatasetID = $datasetsResponse.datasets[0].id
        Write-Host "`n📌 Using dataset: $DatasetID" -ForegroundColor Cyan
    }
}
catch {
    Write-Host "❌ Failed to list datasets: $_" -ForegroundColor Red
    exit 1
}

# Step 2: Get dataset details with items
Write-Host "`n📦 Step 2: Fetching dataset details..." -ForegroundColor Yellow

try {
    $datasetResponse = Invoke-RestMethod -Uri "$GatewayUrl/api/datasets/$DatasetID/items" `
        -Method GET `
        -ContentType "application/json"
    
    $dataset = $datasetResponse.dataset
    Write-Host "✅ Dataset retrieved:" -ForegroundColor Green
    Write-Host "   Title: $($dataset.title)" -ForegroundColor White
    Write-Host "   Items: $($dataset.items.Count)" -ForegroundColor Cyan
    Write-Host "   URL: $($dataset.url)" -ForegroundColor Gray
}
catch {
    Write-Host "❌ Failed to get dataset: $_" -ForegroundColor Red
    exit 1
}

# Step 3: Publish dataset to dane.gov.pl
Write-Host "`n🚀 Step 3: Publishing dataset to dane.gov.pl..." -ForegroundColor Yellow

try {
    $publishResponse = Invoke-RestMethod -Uri "$GatewayUrl/api/datasets/$DatasetID/publish" `
        -Method POST `
        -ContentType "application/json"
    
    Write-Host "✅ Dataset queued for publication!" -ForegroundColor Green
    Write-Host "   Dataset ID: $($publishResponse.dataset_id)" -ForegroundColor Cyan
    Write-Host "   Status: $($publishResponse.status)" -ForegroundColor Yellow
    Write-Host "   Message: $($publishResponse.message)" -ForegroundColor Gray
}
catch {
    Write-Host "❌ Failed to publish dataset: $_" -ForegroundColor Red
    if ($_.ErrorDetails.Message) {
        Write-Host "   Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
    }
    exit 1
}

# Step 4: Instructions for monitoring
Write-Host "`n👀 Step 4: Monitoring publication..." -ForegroundColor Yellow
Write-Host "   The dataset is now being processed by the publisher service." -ForegroundColor Gray
Write-Host "   Check the publisher logs for progress:" -ForegroundColor Gray
Write-Host "" 
Write-Host "   docker logs odnalezione-publisher -f" -ForegroundColor Cyan
Write-Host ""

# Summary
Write-Host "`n" + ("=" * 70) -ForegroundColor DarkGray
Write-Host "✅ Pipeline Test Complete" -ForegroundColor Green
Write-Host ("=" * 70) -ForegroundColor DarkGray
Write-Host ""
Write-Host "Pipeline Flow:" -ForegroundColor White
Write-Host "  1️⃣  Gateway API receives publish request" -ForegroundColor Gray
Write-Host "  2️⃣  Gateway fetches dataset from PostgreSQL" -ForegroundColor Gray
Write-Host "  3️⃣  Gateway publishes 'dataset.publish' event to RabbitMQ" -ForegroundColor Gray
Write-Host "  4️⃣  Publisher consumes event from queue" -ForegroundColor Gray
Write-Host "  5️⃣  Publisher creates dataset on dane.gov.pl via API" -ForegroundColor Gray
Write-Host "  6️⃣  Publisher publishes 'dataset.published' success event" -ForegroundColor Gray
Write-Host ""
Write-Host "Next Steps:" -ForegroundColor White
Write-Host "  • Check dane.gov.pl mock API at http://localhost:8000" -ForegroundColor Gray
Write-Host "  • View RabbitMQ management at http://localhost:15674" -ForegroundColor Gray
Write-Host "  • Monitor publisher logs for confirmation" -ForegroundColor Gray
Write-Host ""
