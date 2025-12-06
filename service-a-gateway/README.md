# Service A: Gateway (Go + HTMX + Tailwind)

Frontend gateway service for the "Odnalezione Zguby" system. Handles user interface, form submissions, and image uploads.

## 🎯 Overview

Service A provides:
- **HTMX-powered UI** for creating and browsing lost items
- **AI-powered image analysis** using Vision API (GPT-4o)
- **Image upload** to MinIO S3-compatible storage
- **Event publishing** to RabbitMQ for downstream processing


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

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | Home page |
| GET | `/create` | Create form |
| POST | `/create` | Submit new lost item |
| GET | `/browse` | Browse all items |
| GET | `/zguba/{id}` | View item details |

### API Routes

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/analyze-image` | Analyze image with AI (JSON) |
| POST | `/api/analyze-image-form` | Analyze image (HTMX partial) |
| GET | `/api/health` | Health check |
| GET | `/health` | Health check |

## 🎨 UI Features

### Create Form
- Drag & drop image upload
- Real-time image preview
- AI-powered description generation
- Auto-fill suggestions
- Form validation
- HTMX for smooth interactions

### Browse Page
- Grid/list view toggle
- Real-time search filtering
- Category filtering
- Status filtering
- Responsive card layout

### Detail Page
- Full item information
- Image viewer
- Processing timeline
- Status tracking

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
  "found_date": "2024-01-15T00:00:00Z",
  "image_url": "http://localhost:9000/lost-items-images/uploads/...",
  "contact_info": "biuro@urzad.pl",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### MinIO Storage

Images uploaded to: `lost-items-images/uploads/{date}/{uuid}.{ext}`

Public URL format: `http://localhost:9000/lost-items-images/uploads/...`

### Vision API

Requests OpenAI GPT-4o to analyze images and return:
- Detailed description (Polish)
- Category suggestion
- Confidence level

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

- [x] HTMX-powered reactive UI
- [x] Tailwind CSS styling
- [x] Image upload to MinIO
- [x] AI image analysis with GPT-4o
- [x] RabbitMQ event publishing
- [x] Form validation
- [x] Real-time search/filtering
- [x] Responsive design
- [x] Health checks
- [x] Structured logging
- [x] Graceful shutdown

### 🚧 Future Enhancements

- [ ] User authentication
- [ ] Database persistence (currently in-memory)
- [ ] Image optimization/thumbnails
- [ ] Rate limiting
- [ ] CSRF protection
- [ ] Internationalization (i18n)
- [ ] Analytics/metrics
- [ ] Admin dashboard

## 🤝 Integration with Other Services

### Service B (AI Worker)
Consumes `item.submitted` events from queue `q.lost-items.ingest`

### Service C (Publisher)
Receives processed items via queue `q.lost-items.publish`

## 📚 Technologies Used

- **Go** - Backend language
- **Gorilla Mux** - HTTP router
- **HTMX** - Dynamic UI without JavaScript frameworks
- **Tailwind CSS** - Utility-first CSS
- **Alpine.js** - Minimal JavaScript for interactions
- **MinIO Go Client** - S3-compatible storage
- **RabbitMQ Go Client** - Message queue
- **Zerolog** - Structured logging

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
