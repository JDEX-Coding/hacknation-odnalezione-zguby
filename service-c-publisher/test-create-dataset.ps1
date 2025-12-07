#!/usr/bin/env pwsh
# Test script for creating a dataset on dane.gov.pl API
# This script demonstrates the complete flow: login -> create dataset -> create resource

param(
    [string]$ApiUrl = "http://localhost:8000",
    [string]$Email = "admin2@mcod.local",
    [string]$Password = "Hacknation-2025"
)

Write-Host "`n🔐 Step 1: Logging in to dane.gov.pl API..." -ForegroundColor Cyan

# Login request (JSON:API format)
$loginPayload = @{
    data = @{
        type = "login"
        attributes = @{
            email = $Email
            password = $Password
        }
    }
} | ConvertTo-Json -Depth 10

try {
    $loginResponse = Invoke-RestMethod -Uri "$ApiUrl/auth/login" `
        -Method POST `
        -ContentType "application/json" `
        -Body $loginPayload

    $token = $loginResponse.data.attributes.token
    Write-Host "✅ Login successful!" -ForegroundColor Green
    Write-Host "   Token: $($token.Substring(0, 50))..." -ForegroundColor Gray
}
catch {
    Write-Host "❌ Login failed: $_" -ForegroundColor Red
    exit 1
}

Write-Host "`n📦 Step 2: Creating dataset..." -ForegroundColor Cyan

# Dataset creation request (JSON:API format)
$datasetPayload = @{
    data = @{
        type = "dataset"
        attributes = @{
            title = "Rzeczy Znalezione - System Testowy 2025"
            notes = "Zbiór danych o rzeczach znalezionych w mieście, zgłoszonych przez mieszkańców. Zawiera opisy przedmiotów, lokalizacje znalezienia oraz dane kontaktowe."
            url = "https://example.com/lost-items"
            category = "transport"
            update_frequency = "daily"
            license_condition_db_or_copyrighted = $false
            license_condition_modification = $true
            license_condition_responsibilities = $false
            license_condition_personal_data = $false
            license_condition_source = $true
            status = "published"
            tags = @(
                @{
                    name = "rzeczy znalezione"
                    display_name = "rzeczy znalezione"
                },
                @{
                    name = "biuro rzeczy znalezionych"
                    display_name = "biuro rzeczy znalezionych"
                },
                @{
                    name = "zguby"
                    display_name = "zguby"
                }
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

try {
    $datasetResponse = Invoke-RestMethod -Uri "$ApiUrl/api/datasets" `
        -Method POST `
        -Headers @{ 
            "Authorization" = "Bearer $token"
            "Content-Type" = "application/json"
            "Accept" = "application/json"
        } `
        -Body $datasetPayload

    $datasetId = $datasetResponse.data.id
    Write-Host "✅ Dataset created successfully!" -ForegroundColor Green
    Write-Host "   Dataset ID: $datasetId" -ForegroundColor Gray
    Write-Host "   Title: $($datasetResponse.data.attributes.title)" -ForegroundColor Gray
}
catch {
    Write-Host "❌ Dataset creation failed: $_" -ForegroundColor Red
    Write-Host "   Response: $($_.Exception.Response)" -ForegroundColor Red
    if ($_.ErrorDetails.Message) {
        Write-Host "   Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
    }
    exit 1
}

Write-Host "`n📄 Step 3: Creating resource (file/data)..." -ForegroundColor Cyan

# Resource creation request (JSON:API format)
$resourcePayload = @{
    data = @{
        type = "resource"
        attributes = @{
            title = "Dane CSV - Rzeczy Znalezione 2025"
            description = "Plik CSV zawierający szczegółowe dane o rzeczach znalezionych, w tym opisy, daty znalezienia, lokalizacje i kategorie przedmiotów."
            format = "CSV"
            link = "https://example.com/dane-rzeczy-znalezione.csv"
            openness_score = 3
            file_info = @{
                size = 524288  # 512 KB example
                encoding = "UTF-8"
            }
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

try {
    $resourceResponse = Invoke-RestMethod -Uri "$ApiUrl/api/resources" `
        -Method POST `
        -Headers @{ 
            "Authorization" = "Bearer $token"
            "Content-Type" = "application/json"
            "Accept" = "application/json"
        } `
        -Body $resourcePayload

    $resourceId = $resourceResponse.data.id
    Write-Host "✅ Resource created successfully!" -ForegroundColor Green
    Write-Host "   Resource ID: $resourceId" -ForegroundColor Gray
    Write-Host "   Title: $($resourceResponse.data.attributes.title)" -ForegroundColor Gray
    Write-Host "   Format: $($resourceResponse.data.attributes.format)" -ForegroundColor Gray
}
catch {
    Write-Host "❌ Resource creation failed: $_" -ForegroundColor Red
    Write-Host "   Response: $($_.Exception.Response)" -ForegroundColor Red
    if ($_.ErrorDetails.Message) {
        Write-Host "   Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
    }
    exit 1
}

Write-Host "`n🎉 Complete! Dataset and resource created successfully." -ForegroundColor Green
Write-Host "   View dataset at: $ApiUrl/dataset/$datasetId" -ForegroundColor Cyan
Write-Host ""
