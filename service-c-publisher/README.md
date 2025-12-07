# Service C: Publisher

Publisher service that publishes lost items as resources to dane.gov.pl dataset.

## Overview

Service C listens to the `item.vectorized` event from RabbitMQ and publishes each lost item as a **resource** (file/entry) to an existing dataset on dane.gov.pl.

## Features

- ✅ **POST-only API**: All requests use POST (login, create dataset, add resources)
- ✅ **Bearer Token Auth**: Automatic login and token management
- ✅ **Resource Publishing**: Adds each item as a resource to a dataset
- ✅ **Auto Dataset Creation**: Optional automatic dataset creation
- ✅ **RabbitMQ Integration**: Consumes `item.vectorized`, publishes `item.published`
- ✅ **Graceful Shutdown**: Proper cleanup and error handling

## Architecture

```
RabbitMQ Queue              Publisher Service           dane.gov.pl API
q.lost-items.publish  -->   Consumer                --> POST /auth/login (get token)
(item.vectorized)      -->   ├─ Login & Auth        --> POST /api/datasets (optional)
                            ├─ Resource Formatter   --> POST /api/datasets/{id}/resources
                            └─ API Client (Bearer)  
                                 │
                                 ├─ Success --> Publish item.published
                                 └─ Error   --> Nack & Requeue
```

## Configuration

### Environment Variables

Create a `.env` file based on `.env.example`:

#### Required Settings

```env
# dane.gov.pl API credentials
DANE_GOV_API_URL=http://localhost:8000
DANE_GOV_EMAIL=admin@mcod.local
DANE_GOV_PASSWORD=your_password_here

# Dataset configuration
DATASET_ID=abc-123-def               # Required if AUTO_CREATE_DATASET=false
AUTO_CREATE_DATASET=false            # Set to true to auto-create dataset
```

#### Optional Settings

```env
# RabbitMQ
RABBITMQ_URL=amqp://admin:admin123@localhost:5672/
RABBITMQ_EXCHANGE=lost-found.events
RABBITMQ_QUEUE=q.lost-items.publish
RABBITMQ_ROUTING_KEY=item.vectorized

# Publisher info
PUBLISHER_NAME=Urząd Miasta - System Rzeczy Znalezionych
PUBLISHER_ID=org-001
BASE_URL=http://localhost:8080
```

## Workflow

### 1. Startup & Authentication

```
1. Load config from .env
2. Connect to RabbitMQ
3. POST /auth/login → Receive Bearer token
4. If AUTO_CREATE_DATASET=true and DATASET_ID empty:
   - POST /api/datasets → Create dataset
   - Save dataset ID for resources
```

### 2. Message Processing

```
For each item.vectorized event:
1. Format item to ResourceRequest
2. POST /api/datasets/{dataset_id}/resources
   - Headers: Authorization: Bearer {token}
   - Body: Item data + image URL + metadata
3. Publish item.published event
4. ACK message
```

### 3. API Requests (POST Only)

All requests use **POST with Bearer token**:

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "data": {
    "type": "login",
    "attributes": {
      "email": "admin@mcod.local",
      "password": "password123"
    }
  }
}
```

#### Add Resource (main workflow)
```http
POST /api/datasets/{dataset_id}/resources
Authorization: Bearer {token}
Content-Type: application/json

{
  "data": {
    "type": "resource",
    "attributes": {
      "name": "Elektronika - Telefon Samsung",
      "description": "Znaleziony w parku...",
      "format": "JPG",
      "url": "http://minio:9000/images/abc123.jpg",
      "custom_fields": {
        "item_id": "abc123",
        "found_date": "2025-01-15",
        "location": "Park Centralny"
      }
    }
  }
}
```

## Running Locally

```bash
# Copy and configure .env
cp .env.example .env
# Edit .env with your credentials

# Install dependencies
go mod download

# Run the service
go run main.go
```

## Troubleshooting

### "Failed to login"
- Check `DANE_GOV_EMAIL` and `DANE_GOV_PASSWORD` in `.env`
- Verify API URL is correct

### "DATASET_ID is required"
- Set `DATASET_ID` in `.env`, or
- Set `AUTO_CREATE_DATASET=true` to create automatically

### "Failed to add resource"
- Verify Bearer token is valid
- Ensure dataset exists and ID is correct
