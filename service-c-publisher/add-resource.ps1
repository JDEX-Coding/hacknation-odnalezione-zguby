#!/usr/bin/env pwsh
# Quick script to add a resource to an existing dataset
# Usage: .\add-resource.ps1 -Token "your-jwt-token" -DatasetId 123

param(
    [Parameter(Mandatory=$true)]
    [string]$Token,
    
    [Parameter(Mandatory=$true)]
    [string]$DatasetId,
    
    [string]$ApiUrl = "http://localhost:8000",
    [string]$Title = "Dane 2025",
    [string]$Description = "Plik CSV z danymi o rzeczach znalezionych",
    [string]$Format = "CSV",
    [string]$FileUrl = "https://example.com/dane.csv"
)

Write-Host "`n📄 Creating resource for dataset $DatasetId..." -ForegroundColor Cyan

# Resource payload in JSON:API format
$resource = @{
    data = @{
        type = "resource"
        attributes = @{
            title = $Title
            description = $Description
            format = $Format
            link = $FileUrl
            openness_score = 3
        }
        relationships = @{
            dataset = @{
                data = @{
                    type = "dataset"
                    id = $DatasetId
                }
            }
        }
    }
} | ConvertTo-Json -Depth 10

Write-Host "Request payload:" -ForegroundColor Gray
Write-Host $resource -ForegroundColor DarkGray
Write-Host ""

try {
    $response = Invoke-RestMethod -Uri "$ApiUrl/api/resources" `
        -Method POST `
        -Headers @{ 
            "Authorization" = "Bearer $Token"
            "Content-Type" = "application/json"
            "Accept" = "application/json"
        } `
        -Body $resource

    Write-Host "✅ Resource created successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Response:" -ForegroundColor Gray
    $response | ConvertTo-Json -Depth 10 | Write-Host -ForegroundColor DarkGray
    Write-Host ""
    Write-Host "Resource ID: $($response.data.id)" -ForegroundColor Cyan
}
catch {
    Write-Host "❌ Failed to create resource" -ForegroundColor Red
    Write-Host "Error: $_" -ForegroundColor Red
    
    if ($_.ErrorDetails.Message) {
        Write-Host "`nAPI Response:" -ForegroundColor Yellow
        $_.ErrorDetails.Message | ConvertFrom-Json | ConvertTo-Json -Depth 10 | Write-Host -ForegroundColor DarkYellow
    }
    
    exit 1
}
