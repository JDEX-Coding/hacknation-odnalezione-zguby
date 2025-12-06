# PowerShell script to set up CLIP service for local development
# Run this from the clip-service directory

Write-Host "🚀 Setting up CLIP Service for Local Development" -ForegroundColor Cyan
Write-Host ""

# Check Python version
Write-Host "Checking Python version..." -ForegroundColor Yellow
$pythonVersion = python --version 2>&1
Write-Host "Found: $pythonVersion" -ForegroundColor Green
Write-Host ""

# Create virtual environment if it doesn't exist
if (-not (Test-Path "venv")) {
    Write-Host "Creating virtual environment..." -ForegroundColor Yellow
    python -m venv venv
    Write-Host "✅ Virtual environment created" -ForegroundColor Green
} else {
    Write-Host "✅ Virtual environment already exists" -ForegroundColor Green
}
Write-Host ""

# Activate virtual environment
Write-Host "Activating virtual environment..." -ForegroundColor Yellow
& .\venv\Scripts\Activate.ps1

# Upgrade pip
Write-Host "Upgrading pip..." -ForegroundColor Yellow
python -m pip install --upgrade pip
Write-Host ""

# Install dependencies
Write-Host "Installing dependencies (CPU-only PyTorch for faster install)..." -ForegroundColor Yellow
pip install -r requirements-dev.txt
Write-Host "✅ Dependencies installed" -ForegroundColor Green
Write-Host ""

# Create .env file if it doesn't exist
if (-not (Test-Path ".env")) {
    Write-Host "Creating .env file from .env.local..." -ForegroundColor Yellow
    Copy-Item .env.local .env
    Write-Host "✅ .env file created" -ForegroundColor Green
} else {
    Write-Host "⚠️  .env file already exists, skipping..." -ForegroundColor Yellow
}
Write-Host ""

# Create cache directory
Write-Host "Creating cache directory..." -ForegroundColor Yellow
$cacheDir = ".cache\huggingface"
if (-not (Test-Path $cacheDir)) {
    New-Item -ItemType Directory -Path $cacheDir -Force | Out-Null
    Write-Host "✅ Cache directory created" -ForegroundColor Green
} else {
    Write-Host "✅ Cache directory already exists" -ForegroundColor Green
}
Write-Host ""

# Pre-download CLIP model (optional but recommended)
$downloadModel = Read-Host "Do you want to pre-download the CLIP model now? (y/N)"
if ($downloadModel -eq 'y' -or $downloadModel -eq 'Y') {
    Write-Host "Downloading CLIP model (this may take a few minutes)..." -ForegroundColor Yellow
    python -c "from transformers import CLIPProcessor, CLIPModel; CLIPProcessor.from_pretrained('openai/clip-vit-base-patch32'); CLIPModel.from_pretrained('openai/clip-vit-base-patch32'); print('✅ Model downloaded successfully')"
    Write-Host ""
}

Write-Host "🎉 Setup complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "1. Make sure RabbitMQ is running (docker-compose up rabbitmq -d)" -ForegroundColor White
Write-Host "2. Make sure MinIO is running (docker-compose up minio -d)" -ForegroundColor White
Write-Host "3. Run the service: python main.py" -ForegroundColor White
Write-Host ""
Write-Host "Or run everything with: docker-compose up rabbitmq minio -d" -ForegroundColor Yellow
