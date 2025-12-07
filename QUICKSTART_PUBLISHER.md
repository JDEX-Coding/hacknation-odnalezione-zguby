# 🚀 Quick Start: Testing Service C (Publisher)

This guide helps you quickly test the complete Publisher service end-to-end.

## Prerequisites

- Docker & Docker Compose installed
- RabbitMQ running (from docker-compose)
- (Optional) OpenAI API key for full pipeline test

## Option 1: Test with Mock API (Fastest)

### Step 1: Start Infrastructure

```powershell
# Start RabbitMQ only
docker compose up -d rabbitmq

# Wait for initialization
Start-Sleep -Seconds 10

# Verify RabbitMQ is ready
docker compose logs rabbitmq-init
```

### Step 2: Start Mock dane.gov.pl API

```powershell
# Open a new terminal
cd service-c-publisher
go run mock-api.go
```

You should see:
```
🚀 Mock dane.gov.pl API server starting on :8000
📍 Health check: http://localhost:8000/health
📍 API endpoint: http://localhost:8000/api/v1/datasets
```

### Step 3: Start Publisher Service

```powershell
# Open another terminal
cd service-c-publisher

# Set environment to use mock API with login credentials
$env:DANE_GOV_API_URL="http://localhost:8000"
$env:DANE_GOV_EMAIL="admin@mcod.local"
$env:DANE_GOV_PASSWORD="test-password"
$env:RABBITMQ_URL="amqp://admin:admin123@localhost:5672/"

# Run publisher
go run main.go
```

You should see:
```
🚀 Starting Service C: Publisher
Logging in to dane.gov.pl...
✅ Successfully logged in to dane.gov.pl email=admin@mcod.local
✅ dane.gov.pl API is healthy
✅ Publisher service initialized successfully
🎧 Listening for messages on RabbitMQ...
```

### Step 4: Send Test Event

```powershell
# Option A: Using event-emulator (if available)
cd event-emulator
go run emulator/event_emulator.go

# Option B: Using RabbitMQ Management UI
# 1. Open http://localhost:15672 (admin/admin123)
# 2. Go to "Queues" tab
# 3. Click "q.lost-items.publish"
# 4. Scroll to "Publish message"
# 5. Paste this JSON in payload:
```

```json
{
  "id": "test-123",
  "title": "Znaleziony portfel",
  "description": "Czarny portfel skórzany znaleziony w parku",
  "category": "Portfele i torby",
  "location": "Park Łazienkowski, Warszawa",
  "found_date": "2024-12-06T10:00:00Z",
  "reporting_date": "2024-12-06T10:30:00Z",
  "reporting_location": "Urząd Dzielnicy Śródmieście",
  "image_url": "http://localhost:9000/lost-items-images/wallet.jpg",
  "contact_email": "biuro@warszawa.pl",
  "contact_phone": "+48-22-123-4567",
  "timestamp": "2024-12-06T10:30:00Z"
}
```

**Set routing key to:** `item.vectorized`

### Step 5: Observe the Flow

**Publisher Terminal:**
```
📨 Received message routing_key=item.vectorized message_id=test-123
📝 Processing item for publication item_id=test-123 title="Znaleziony portfel"
✅ Successfully published item to dane.gov.pl dataset_id=abc-xyz-... url=http://localhost:8000/datasets/...
✅ Successfully processed and published item item_id=test-123 duration_ms=234
```

**Mock API Terminal:**
```
✅ Dataset created: abc-xyz-123 - Znaleziony portfel
```

## Option 2: Test with Docker Compose (Full Stack)

### Step 1: Configure Environment

```powershell
# Copy and edit .env file
Copy-Item .env.example .env
notepad .env

# Set at minimum:
# DANE_GOV_API_URL=http://localhost:8000  # or real API
# DANE_GOV_API_KEY=your-key-here          # if required
```

### Step 2: Start All Services

```powershell
# Build and start everything
docker compose up --build -d

# Watch publisher logs
docker compose logs -f c-publisher
```

### Step 3: Submit an Item via Gateway

```powershell
# Open browser
Start-Process "http://localhost:8080"

# Or use curl
curl -X POST http://localhost:8080/create `
  -F "title=Znaleziony telefon" `
  -F "description=Czarny iPhone 13" `
  -F "category=Telefony" `
  -F "location=Park Łazienkowski" `
  -F "found_date=2024-12-06" `
  -F "contact_email=biuro@warszawa.pl" `
  -F "image=@photo.jpg"
```

### Step 4: Monitor the Pipeline

```powershell
# Watch all logs
docker compose logs -f

# Or specific services
docker compose logs -f a-gateway c-publisher

# Check RabbitMQ queues
Start-Process "http://localhost:15672"  # admin/admin123
```

## Option 3: Integration Test with Real API

### Step 1: Get dane.gov.pl API Credentials

1. Register at https://dane.gov.pl
2. Create an organization
3. Get API key from settings

```powershell
# Set environment variables with your account credentials
$env:DANE_GOV_API_URL="https://api.dane.gov.pl"
$env:DANE_GOV_EMAIL="your-email@example.com"
$env:DANE_GOV_PASSWORD="your-password"
$env:PUBLISHER_ID="your-org-id"
```v:DANE_GOV_API_KEY="your-real-api-key"
$env:PUBLISHER_ID="your-org-id"
```

### Step 3: Run Publisher

```powershell
cd service-c-publisher
go run main.go
```

### Step 4: Verify on dane.gov.pl

1. Login to https://dane.gov.pl
2. Go to your organization
3. Check published datasets
4. Verify item appears with correct metadata

## Troubleshooting

### Publisher can't connect to RabbitMQ
```powershell
# Check RabbitMQ is running
docker ps | Select-String rabbitmq

# Check logs
docker logs odnalezione-rabbitmq

# Test connection
curl -u admin:admin123 http://localhost:15672/api/overview
```

### No messages being received
```powershell
# Check queue exists
curl -u admin:admin123 http://localhost:15672/api/queues/%2F/q.lost-items.publish

# Check bindings
curl -u admin:admin123 http://localhost:15672/api/exchanges/%2F/lost-found.events/bindings/source

# Manually publish test message via UI
Start-Process "http://localhost:15672"
```

### Mock API not responding
```powershell
# Check if port 8000 is free
Test-NetConnection -ComputerName localhost -Port 8000

# Test health check
curl http://localhost:8000/health
```

### Docker build fails
```powershell
# Clean and rebuild
docker compose down
docker system prune -f
docker compose build --no-cache c-publisher
docker compose up -d c-publisher
```

## Success Indicators

✅ **Publisher logs show:**
- "RabbitMQ consumer initialized"
- "Listening for messages"
- Messages received and processed
- "Successfully published item"

✅ **Mock API logs show:**
- "Dataset created: [ID]"

✅ **RabbitMQ UI shows:**
- Queue `q.lost-items.publish` has consumers
- Messages are being acknowledged
- No messages stuck in queue

✅ **dane.gov.pl shows:**
- Dataset appears in organization
- Metadata is correct
- Image is accessible

## Next Steps

- Configure real dane.gov.pl credentials
- Test with full pipeline (Gateway → CLIP → Publisher)
- Set up monitoring and alerting
- Configure error handling and DLQ
- Add deployment automation

## Useful Commands

```powershell
# View all service statuses
docker compose ps

# Restart publisher only
docker compose restart c-publisher

# View real-time logs
docker compose logs -f --tail=50 c-publisher

# Stop everything
docker compose down

# Clean everything
docker compose down -v
docker system prune -af
```
