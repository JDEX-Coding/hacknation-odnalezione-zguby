# System "Odnalezione Zguby" - Integracja z dane.gov.pl

[cite_start]Projekt systemu realizujący wyzwanie hackathonowe, ułatwiający samorządom szybkie (max. 5 kroków) i ustandaryzowane wgrywanie danych o rzeczach znalezionych do portalu **dane.gov.pl**[cite: 12, 20]. [cite_start]System wykorzystuje AI do opisywania zdjęć oraz wektoryzację (Qdrant) dla wyszukiwania semantycznego, spełniając wymóg dostarczania danych w formacie czytelnym maszynowo[cite: 8, 29].

## 🚀 Quick Start

### Prerequisites
- Docker Engine 20.10+
- Docker Compose 2.0+
- At least 4GB RAM available

### Setup in 3 Steps

1. **Clone and initialize environment:**
```bash
git clone <repository-url>
cd hacknation-odnalezione-zguby
make init
```

2. **Edit `.env` file with your API keys:**
```bash
# Required: Add your OpenAI/Vision API key
VISION_API_KEY=your-api-key-here
```

3. **Verify setup:**
```bash
bash verify-setup.sh
```

That's it! All infrastructure services are now running:
- 🐇 **RabbitMQ**: http://localhost:15672 (admin/admin123)
- 🔍 **Qdrant**: http://localhost:6333/dashboard
- 📦 **MinIO**: http://localhost:9001 (minioadmin/minioadmin123)

For detailed Docker documentation, see [DOCKER.md](./DOCKER.md)</text>

<old_text line=58>
4.  **Infrastruktura:**
    -   **RabbitMQ:** Message Broker (Topic Exchange `lost-found.events`).
    -   **Qdrant:** Baza wektorowa.

## 🏗 Architektura Systemu

Architektura oparta jest na mikroserwisach i asynchronicznym przetwarzaniu zdarzeń. Składa się z trzech głównych serwisów:

### Komponenty:

1.  **Service A: Gateway (Go + HTMX)**

    -   **Rola:** Interfejs dla urzędnika (Frontend) i punkt wejścia danych.
    -   **Zadania:**
        -   Obsługa formularza HTMX.
        -   Komunikacja z **Vision API** (np. GPT-4o/LLaVA) w czasie rzeczywistym, aby wygenerować opis przedmiotu na podstawie wgranego zdjęcia (wsparcie UX).
        -   Walidacja wstępna i wysłanie zdarzenia `ItemSubmitted` do RabbitMQ.

2.  **Service B: AI Worker (Python)**

    -   **Rola:** Przetwarzanie semantyczne (Heavy lifting).
    -   **Zadania:**
        -   Konsumpcja zdarzeń z kolejki `q.lost-items.ingest`.
        -   Generowanie embeddingów (wektorów) dla tekstu i obrazu.
        -   Zapis metadanych wektorowych do bazy **Qdrant**.
        -   Emisja zdarzenia `ItemVectorized` do RabbitMQ.

3.  **Service C: Publisher (Go)**

    -   **Rola:** Integracja z API rządowym.
    -   **Zadania:**
        -   Konsumpcja przetworzonych danych z kolejki `q.lost-items.publish`.
        -   [cite_start]Konwersja danych do standardu wymaganego przez dane.gov.pl (JSON-LD / CSV)[cite: 29].
        -   Autoryzacja i wysyłka danych (POST) do API portalu.

4.  **Infrastruktura:**
    -   **RabbitMQ:** Message Broker (Topic Exchange `lost-found.events`).
    -   **Qdrant:** Baza wektorowa.

---

### 📊 Diagram Przepływu

```mermaid
graph TD
    %% Aktorzy i Systemy Zewnetrzne
    User((Urzędnik))
    VisionAPI[External Vision API]
    DaneGov[API dane.gov.pl]
    Qdrant[(Qdrant Vector DB)]

    %% Definicja RabbitMQ
    subgraph RabbitMQ_Cluster["RabbitMQ Broker"]
        Exchange((Exchange: lost-found.events))
        Q_Ingest[Queue: q.lost-items.ingest]
        Q_Publish[Queue: q.lost-items.publish]
    end

    %% Serwis A: Gateway
    subgraph Svc_Gateway["Service A: Gateway Go+HTMX"]
        UI_Handler[HTMX Form Handler]
        Img_Helper[Image Helper]
    end

    %% Serwis B: AI Worker
    subgraph Svc_Python["Service B: Python Worker"]
        Py_Consumer[Event Consumer]
        Vector_Logic[Vector Engine]
    end

    %% Serwis C: Publisher
    subgraph Svc_Publisher["Service C: Publisher Go"]
        Go_Consumer[Event Consumer]
        Data_Formatter[Gov Data Formatter]
    end

    %% Styles
    style RabbitMQ_Cluster fill:#ff9900,stroke:#333,color:#fff
    style Svc_Gateway fill:#00ADD8,stroke:#333,color:#fff
    style Svc_Python fill:#3776AB,stroke:#333,color:#fff
    style Svc_Publisher fill:#00ADD8,stroke:#333,color:#fff

    %% --- RELACJE ---

    %% 1. Interakcja Urzędnika
    User -->|1. Formularz + Zdjęcie| UI_Handler
    UI_Handler -.->|2. Get Description| Img_Helper
    Img_Helper -.->|3. Analyze Img| VisionAPI
    VisionAPI -.->|4. Return Text| Img_Helper
    Img_Helper -.->|5. Fill Form| UI_Handler

    %% 2. Wysyłka do Kolejki Ingest
    UI_Handler -->|6. Submit Event| Exchange
    Exchange -->|key: item.submitted| Q_Ingest

    %% 3. Przetwarzanie Python
    Q_Ingest -->|7. Consume| Py_Consumer
    Py_Consumer --> Vector_Logic
    Vector_Logic -->|8. Upsert| Qdrant

    %% 4. Wysyłka do Kolejki Publish
    Vector_Logic -->|9. Publish Event| Exchange
    Exchange -->|key: item.vectorized| Q_Publish

    %% 5. Publikacja do Gov.pl
    Q_Publish -->|10. Consume| Go_Consumer
    Go_Consumer --> Data_Formatter
    Data_Formatter -->|11. Final POST| DaneGov
```

---

## 🐳 Docker Infrastructure

Wszystkie wymagane serwisy są dostępne poprzez Docker Compose:

### Included Services

| Service | Port(s) | Purpose | Web UI |
|---------|---------|---------|--------|
| **RabbitMQ** | 5672, 15672 | Message broker z pre-configured exchange i queues | http://localhost:15672 |
| **Qdrant** | 6333, 6334 | Vector database dla semantic search | http://localhost:6333/dashboard |
| **MinIO** | 9000, 9001 | S3-compatible storage dla zdjęć | http://localhost:9001 |

### Quick Commands

```bash
# Start all services
make up

# View logs
make logs-follow

# Check service health
make health

# Stop all services
make down

# Full cleanup (removes data)
make clean
```

### Configuration

Pre-configured setup includes:
- **RabbitMQ Exchange**: `lost-found.events` (topic)
- **RabbitMQ Queues**: 
  - `q.lost-items.ingest` (routing key: `item.submitted`)
  - `q.lost-items.publish` (routing key: `item.vectorized`)
- **MinIO Bucket**: `lost-items-images` (public read access)

For complete documentation, see [DOCKER.md](./DOCKER.md)

---

## 📁 Project Structure

```
hacknation-odnalezione-zguby/
├── docker-compose.yml       # Infrastructure services definition
├── .env.example            # Environment variables template
├── Makefile                # Convenience commands
├── verify-setup.sh         # Setup verification script
├── DOCKER.md              # Detailed Docker documentation
├── RabbitMQ.md            # RabbitMQ configuration details
└── README.md              # This file
```

---

## 🔧 Development Workflow

1. **Start infrastructure**: `make up`
2. **Develop Service A (Gateway)**: Connect to RabbitMQ at `localhost:5672`
3. **Develop Service B (AI Worker)**: Connect to RabbitMQ, Qdrant at `localhost:6333`, MinIO at `localhost:9000`
4. **Develop Service C (Publisher)**: Connect to RabbitMQ
5. **Monitor**: Use web UIs for debugging
6. **Stop**: `make down` when done

---

## 🌐 Connection Details

### From Your Applications

```python
# RabbitMQ
RABBITMQ_URL = "amqp://admin:admin123@localhost:5672/"

# Qdrant
QDRANT_HOST = "localhost"
QDRANT_PORT = 6333

# MinIO
MINIO_ENDPOINT = "localhost:9000"
MINIO_ACCESS_KEY = "minioadmin"
MINIO_SECRET_KEY = "minioadmin123"
MINIO_BUCKET = "lost-items-images"
```

```go
// RabbitMQ
conn, _ := amqp.Dial("amqp://admin:admin123@localhost:5672/")

// Qdrant
client, _ := qdrant.NewClient(&qdrant.Config{
    Host: "localhost",
    Port: 6333,
})

// MinIO
minioClient, _ := minio.New("localhost:9000", &minio.Options{
    Creds:  credentials.NewStaticV4("minioadmin", "minioadmin123", ""),
    Secure: false,
})
```

---

## 📚 Additional Documentation

- [DOCKER.md](./DOCKER.md) - Complete Docker setup and troubleshooting guide
- [RabbitMQ.md](./RabbitMQ.md) - RabbitMQ queue configuration details

---

## 🔐 Security Note

⚠️ **Default credentials are for development only!** 

Change all passwords before any production deployment.

---

## 💡 Tips

- Keep management UIs open in browser tabs for real-time monitoring
- Use `make logs-follow` to debug message flow
- MinIO bucket has public read access - perfect for serving images
- All data persists in Docker volumes across restarts
- Use `make help` for all available commands</text>

