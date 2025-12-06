# Service A: Gateway (Go + HTMX + Tailwind)

Serwis frontendowy dla systemu "Odnalezione Zguby". Dostarcza interfejs użytkownika dla urzędników oraz punkt wejścia do całego systemu.

## 🎯 Przegląd

Service A realizuje:

-   **UI oparte na HTMX** - Dynamiczny interfejs do tworzenia i przeglądania rzeczy znalezionych
-   **Analiza AI obrazów** - Integracja z Vision API (GPT-4o) do opisu zdjęć w czasie rzeczywistym
-   **Upload zdjęć** - Zapisanie obrazów w MinIO (S3-compatible storage)
-   **Publikacja zdarzeń** - Wysyłanie `item.submitted` do RabbitMQ

## 🏗️ Architektura

```mermaid
graph TB
    User["👤 Urzędnik"]
    UI["🖥️ HTMX UI<br/>Port 8080"]
    VisionAPI["🤖 Vision API<br/>(GPT-4o)"]
    MinIO["📦 MinIO<br/>S3 Storage"]
    RabbitMQ["🐇 RabbitMQ<br/>Message Broker"]
    CLIP["🐍 Service B<br/>CLIP Worker"]

    User -->|Otwiera formularz| UI
    User -->|Wgrywa zdjęcie| UI
    UI -->|Analiza| VisionAPI
    VisionAPI -->|Opis + sugestie| UI
    UI -->|Zapisuje obraz| MinIO
    UI -->|Publikuje event| RabbitMQ
    RabbitMQ -->|Routing| CLIP

    style User fill:#ddd,stroke:#333,color:#000
    style UI fill:#00ADD8,stroke:#333,color:#fff,stroke-width:2px
    style VisionAPI fill:#purple,stroke:#333,color:#fff
    style MinIO fill:#C72C48,stroke:#333,color:#fff
    style RabbitMQ fill:#ff9900,stroke:#333,color:#fff,stroke-width:2px
    style CLIP fill:#3776AB,stroke:#333,color:#fff
```

## 🚀 Quick Start

### 1. Prerequisites

```bash
# Go 1.20+
go version

# Clone repository
git clone <repo>
cd service-a-gateway
```

### 2. Configure Environment

```bash
# Copy environment template
cp .env.example .env

# Edit .env i add your API keys
VISION_API_KEY=sk-...  # OpenAI API key
RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/
MINIO_ENDPOINT=minio:9000
```

### 3. Install Dependencies

```bash
go mod download
go mod tidy
```

### 4. Run the Service

```bash
# From service-a-gateway directory
go run cmd/server/main.go

# Or build and run
go build -o gateway cmd/server/main.go
./gateway
```

### 5. Access the Application

Otwórz przeglądarkę: **http://localhost:8080**

## 📁 Project Structure

```
service-a-gateway/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── handlers/
│   │   └── handlers.go          # HTTP handlers + HTMX responses
│   ├── models/
│   │   └── item.go              # Data models (LostItem, etc.)
│   ├── services/
│   │   ├── rabbitmq.go          # RabbitMQ publisher
│   │   └── vision.go            # Vision API client (GPT-4o)
│   └── storage/
│       └── minio.go             # MinIO S3 client
├── web/
│   ├── templates/
│   │   ├── base.html            # Base layout
│   │   ├── index.html           # Home page
│   │   ├── create.html          # Create form page
│   │   ├── browse.html          # Browse/search page
│   │   └── detail.html          # Item detail page
│   └── static/                  # Static assets (CSS, JS)
├── go.mod
├── go.sum
├── Dockerfile                   # Docker build
├── .env.example
└── README.md
```

## 📡 API Routes

### Web Pages (HTML)

| Method | Endpoint    | Opis                                 |
| ------ | ----------- | ------------------------------------ |
| GET    | `/`         | Strona główna                        |
| GET    | `/create`   | Formularz tworzenia przedmiotu       |
| POST   | `/create`   | Submit nowego przedmiotu (multipart) |
| GET    | `/browse`   | Przeglądanie wszystkich przedmiotów  |
| GET    | `/item/:id` | Szczegóły przedmiotu                 |

### API Routes (JSON)

| Method | Endpoint                | Opis                            |
| ------ | ----------------------- | ------------------------------- |
| POST   | `/api/analyze-image`    | Analiza zdjęcia (JSON response) |
| POST   | `/api/analyze-image-hx` | Analiza zdjęcia (HTMX partial)  |
| GET    | `/api/health`           | Health check                    |

## 🎨 User Interface

### Strona Główna (/)

-   Informacja o systemie
-   Link do formularza
-   Link do przeglądania

### Formularz Tworzenia (/create)

-   Drag & drop upload zdjęcia
-   Preview zdjęcia
-   Real-time AI analysis (GPT-4o)
-   Auto-fill dla nazwy i kategorii
-   Walidacja formularza
-   HTMX partial responses dla smooth UX

### Przeglądanie (/browse)

-   Grid/list view toggle
-   Real-time search filtering
-   Category filtering
-   Responsive card layout
-   Pagination

### Detail Przedmiotu (/item/:id)

-   Full item info
-   Image viewer
-   Processing status
-   Timeline

## 🔌 Integration

### RabbitMQ Publisher

Service A publikuje zdarzenia `item.submitted` do RabbitMQ:

```go
// models/item.go
type ItemSubmittedEvent struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Category    string    `json:"category"`
    Location    string    `json:"location"`
    FoundDate   time.Time `json:"found_date"`
    ImageURL    string    `json:"image_url"`      // MinIO URL
    ContactInfo string    `json:"contact_info"`
    Timestamp   time.Time `json:"timestamp"`
}
```

### MinIO Storage

Struktura folderów:

```
lost-items-images/
└── uploads/
    └── 2025-12-06/
        ├── 550e8400-e29b-41d4-a716-446655440000.jpg
        └── ...
```

Public URL format:

```
http://localhost:9000/lost-items-images/uploads/2025-12-06/550e8400-e29b-41d4-a716-446655440000.jpg
```

### Vision API (GPT-4o)

Request structure:

```json
{
    "model": "gpt-4o",
    "messages": [
        {
            "role": "user",
            "content": [
                {
                    "type": "text",
                    "text": "Opisz ten przedmiot. Odpowiedź w polskim języku."
                },
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "data:image/jpeg;base64,..."
                    }
                }
            ]
        }
    ]
}
```

Response:

```json
{
    "choices": [
        {
            "message": {
                "content": "Na zdjęciu widać czarny portfel skórzany ze złotą klamrą..."
            }
        }
    ]
}
```

## 🧪 Testing

### Manual Testing

```bash
# 1. Start all infrastructure
docker-compose up -d

# 2. Start this service
go run cmd/server/main.go

# 3. Open http://localhost:8080

# 4. Test upload and AI analysis
```

### Health Check

```bash
curl http://localhost:8080/api/health | jq
```

Expected:

```json
{
    "status": "healthy",
    "checks": {
        "storage": "ok",
        "rabbitmq": "ok",
        "vision": "ok"
    }
}
```

## 🐛 Troubleshooting

### Port Already in Use

```bash
# Change port in .env
GATEWAY_PORT=8081

# Then access at http://localhost:8081
```

### RabbitMQ Connection Failed

```bash
# Verify RabbitMQ is running
docker-compose ps rabbitmq

# Check logs
docker-compose logs rabbitmq

# Restart if needed
docker-compose restart rabbitmq
```

### MinIO Upload Failed

```bash
# Verify MinIO is running
docker-compose ps minio

# Verify bucket exists and is public
docker exec -it odnalezione-minio mc ls myminio/

# Recreate bucket if needed
docker-compose up minio-init
```

### Vision API Error

```bash
# Check if API key is set
echo $VISION_API_KEY

# Test API key manually
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $VISION_API_KEY"
```

### Templates Not Found

```bash
# Ensure running from correct directory
cd service-a-gateway

# Or set TEMPLATES_PATH
export TEMPLATES_PATH=web/templates
go run cmd/server/main.go
```

## 🔧 Development

### Hot Reload (Optional)

Install Air:

```bash
go install github.com/cosmtrek/air@latest

# Create .air.toml if needed
air init

# Run with hot reload
air
```

### Build for Production

```bash
# Build binary
go build -o gateway cmd/server/main.go

# Run
./gateway
```

### Docker Build

```bash
# Build Docker image
docker build -t service-a-gateway .

# Run container
docker run -p 8080:8080 --env-file .env service-a-gateway
```

## 📊 Monitoring

### Logs

Service uses structured logging (zerolog):

```bash
# Logs include:
# - HTTP requests with timing
# - Image uploads to MinIO
# - RabbitMQ publishing status
# - Vision API calls and responses
# - Errors with full stack traces
```

### Metrics

Check service health:

```bash
curl http://localhost:8080/api/health | jq
```

Response:

```json
{
    "status": "healthy",
    "checks": {
        "storage": "ok",
        "rabbitmq": "ok",
        "vision": "ok"
    },
    "uptime_seconds": 3600
}
```

## 🌟 Features

### ✅ Implemented

-   [x] HTMX-powered reactive UI bez JavaScript frameworks
-   [x] Tailwind CSS styling (utility-first)
-   [x] Image upload to MinIO S3
-   [x] Real-time AI analysis (GPT-4o Vision API)
-   [x] RabbitMQ event publishing (item.submitted)
-   [x] Form validation (client + server side)
-   [x] Real-time search/filtering HTMX
-   [x] Responsive design (mobile-friendly)
-   [x] Health checks
-   [x] Structured logging (zerolog)
-   [x] Graceful shutdown

### 🚧 Future Enhancements

-   [ ] User authentication + authorization
-   [ ] Database persistence (currently in-memory)
-   [ ] Image optimization/thumbnails
-   [ ] Rate limiting per IP
-   [ ] CSRF protection
-   [ ] Internationalization (i18n) - English support
-   [ ] Analytics/metrics
-   [ ] Admin dashboard
-   [ ] Batch uploads
-   [ ] Search history
-   [ ] Saved items/favorites

## 🤝 Integracja z Innymi Serwisami

### Service B (CLIP Worker) 🐍

-   Konsumuje: `item.submitted` events z `q.lost-items.ingest`
-   Generuje: embeddings dla tekstu i obrazu
-   Publikuje: `item.vectorized` do `q.lost-items.publish`

### Service C (Publisher) 🚀

-   Konsumuje: `item.vectorized` events z `q.lost-items.publish`
-   Konwertuje: do standardu DCAT-AP PL
-   Wysyła: do dane.gov.pl API

### RabbitMQ 🐇

-   Exchange: `lost-found.events` (topic)
-   Queue: `q.lost-items.ingest` (routing_key: `item.submitted`)

### MinIO 📦

-   Bucket: `lost-items-images`
-   Path: `uploads/YYYY-MM-DD/uuid.jpg`
-   Access: Public read for browser display

## 📚 Technologies Used

| Tech                | Purpose                                  |
| ------------------- | ---------------------------------------- |
| **Go 1.20+**        | Backend language                         |
| **Gorilla Mux**     | HTTP router                              |
| **HTMX**            | Dynamic UI updates without JS frameworks |
| **Tailwind CSS**    | Utility-first CSS framework              |
| **Alpine.js**       | Lightweight DOM interactions             |
| **MinIO Go SDK**    | S3-compatible storage                    |
| **RabbitMQ Go SDK** | Message queue                            |
| **Zerolog**         | Structured logging                       |
| **Docker**          | Containerization                         |

## 📄 License

Część projektu Odnalezione Zguby - HackNation

## 🆘 Support

W razie problemów:

1. Sprawdź główny README projektu
2. Sprawdź setup Dockera w `../README.md`
3. Przejrzyj logi serwisu
4. Sprawdź health endpoints
5. Sprawdź RabbitMQ UI (http://localhost:15672)
6. Sprawdź MinIO UI (http://localhost:9001)

---

**Service A: Gateway** - Frontend i punkt wejścia danych dla systemu Odnalezione Zguby

### 2. Configure Environment

```bash
# Copy environment template
cp .env.example .env

# Edit .env and add your OpenAI API key
# VISION_API_KEY=sk-...
```

### 3. Install Dependencies

```bash
go mod download
go mod tidy
```

### 4. Run the Service

```bash
# From service-a-gateway directory
go run cmd/server/main.go

# Or build and run
go build -o gateway cmd/server/main.go
./gateway
```

### 5. Access the Application

Open your browser: **http://localhost:8080**

## 📁 Project Structure

```
service-a-gateway/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── handlers/
│   │   └── handlers.go          # HTTP handlers
│   ├── models/
│   │   └── item.go              # Data models
│   ├── services/
│   │   ├── rabbitmq.go          # RabbitMQ publisher
│   │   └── vision.go            # Vision API client
│   └── storage/
│       └── minio.go             # MinIO storage client
├── web/
│   ├── templates/
│   │   ├── base.html            # Base layout
│   │   ├── index.html           # Home page
│   │   ├── create.html          # Create form
│   │   ├── browse.html          # Browse list
│   │   └── detail.html          # Item detail
│   └── static/                  # Static assets (if any)
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## 📡 API Endpoints

### Web Pages

| Method | Endpoint      | Description          |
| ------ | ------------- | -------------------- |
| GET    | `/`           | Home page            |
| GET    | `/create`     | Create form          |
| POST   | `/create`     | Submit new lost item |
| GET    | `/browse`     | Browse all items     |
| GET    | `/zguba/{id}` | View item details    |

### API Routes

| Method | Endpoint                  | Description                  |
| ------ | ------------------------- | ---------------------------- |
| POST   | `/api/analyze-image`      | Analyze image with AI (JSON) |
| POST   | `/api/analyze-image-form` | Analyze image (HTMX partial) |
| GET    | `/api/health`             | Health check                 |
| GET    | `/health`                 | Health check                 |

## 🎨 UI Features

### Create Form

-   Drag & drop image upload
-   Real-time image preview
-   AI-powered description generation
-   Auto-fill suggestions
-   Form validation
-   HTMX for smooth interactions

### Browse Page

-   Grid/list view toggle
-   Real-time search filtering
-   Category filtering
-   Status filtering
-   Responsive card layout

### Detail Page

-   Full item information
-   Image viewer
-   Processing timeline
-   Status tracking

## 🔌 Integration

### RabbitMQ Events

Service A publishes `item.submitted` events:

```json
{
    "id": "uuid",
    "title": "Found wallet",
    "description": "Black leather wallet...",
    "category": "Portfele i torby",
    "location": "Rynek Główny, Kraków",
    "found_date": "2025-01-15T00:00:00Z",
    "image_url": "http://localhost:9000/lost-items-images/uploads/...",
    "contact_info": "biuro@urzad.pl",
    "timestamp": "2025-01-15T10:30:00Z"
}
```

### MinIO Storage

Images uploaded to: `lost-items-images/uploads/{date}/{uuid}.{ext}`

Public URL format: `http://localhost:9000/lost-items-images/uploads/...`

### Vision API

Requests OpenAI GPT-4o to analyze images and return:

-   Detailed description (Polish)
-   Category suggestion
-   Confidence level

## 🧪 Testing

### Manual Testing

```bash
# 1. Start the service
go run cmd/server/main.go

# 2. Open browser to http://localhost:8080

# 3. Test image upload and AI analysis
```

### Health Check

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{
    "status": "healthy",
    "checks": {
        "storage": "ok",
        "rabbitmq": "ok",
        "vision": "ok"
    }
}
```

## 🐛 Troubleshooting

### Port Already in Use

```bash
# Change port in .env
GATEWAY_PORT=8081
```

### RabbitMQ Connection Failed

```bash
# Verify RabbitMQ is running
docker-compose ps rabbitmq

# Check logs
docker-compose logs rabbitmq

# Restart if needed
docker-compose restart rabbitmq
```

### MinIO Upload Failed

```bash
# Verify MinIO is running
docker-compose ps minio

# Verify bucket exists
docker exec -it odnalezione-minio mc ls myminio/

# Recreate bucket if needed
docker-compose up minio-init
```

### Vision API Error

```bash
# Check API key is set
echo $VISION_API_KEY

# Test API key manually
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $VISION_API_KEY"
```

### Templates Not Found

```bash
# Ensure you're running from the service directory
cd service-a-gateway

# Or set TEMPLATES_PATH
export TEMPLATES_PATH=web/templates
```

## 🔧 Development

### Hot Reload (Optional)

Install Air for hot reload:

```bash
go install github.com/cosmtrek/air@latest

# Create .air.toml if needed
air init

# Run with hot reload
air
```

### Build for Production

```bash
# Build binary
go build -o gateway cmd/server/main.go

# Run
./gateway
```

### Docker Build (Optional)

```bash
# Build Docker image
docker build -t service-a-gateway .

# Run container
docker run -p 8080:8080 --env-file .env service-a-gateway
```

## 📊 Monitoring

### Logs

Service uses structured logging (zerolog):

```bash
# View logs with timestamp
go run cmd/server/main.go

# Logs include:
# - HTTP requests
# - Image uploads
# - RabbitMQ publishing
# - Vision API calls
# - Errors and warnings
```

### Metrics

Check service health:

```bash
curl http://localhost:8080/health | jq
```

## 🌟 Features

### ✅ Implemented

-   [x] HTMX-powered reactive UI
-   [x] Tailwind CSS styling
-   [x] Image upload to MinIO
-   [x] AI image analysis with GPT-4o
-   [x] RabbitMQ event publishing
-   [x] Form validation
-   [x] Real-time search/filtering
-   [x] Responsive design
-   [x] Health checks
-   [x] Structured logging
-   [x] Graceful shutdown

### 🚧 Future Enhancements

-   [ ] User authentication
-   [ ] Database persistence (currently in-memory)
-   [ ] Image optimization/thumbnails
-   [ ] Rate limiting
-   [ ] CSRF protection
-   [ ] Internationalization (i18n)
-   [ ] Analytics/metrics
-   [ ] Admin dashboard

## 🤝 Integration with Other Services

### Service B (AI Worker)

Consumes `item.submitted` events from queue `q.lost-items.ingest`

### Service C (Publisher)

Receives processed items via queue `q.lost-items.publish`

## 📚 Technologies Used

-   **Go** - Backend language
-   **Gorilla Mux** - HTTP router
-   **HTMX** - Dynamic UI without JavaScript frameworks
-   **Tailwind CSS** - Utility-first CSS
-   **Alpine.js** - Minimal JavaScript for interactions
-   **MinIO Go Client** - S3-compatible storage
-   **RabbitMQ Go Client** - Message queue
-   **Zerolog** - Structured logging

## 📄 License

Part of the Odnalezione Zguby system - HackNation project

## 🆘 Support

For issues or questions:

1. Check the main project README
2. Check Docker infrastructure setup in `../DOCKER.md`
3. Review service logs
4. Check health endpoints

---

**Service A: Gateway** - User interface and data ingestion for the Odnalezione Zguby system
