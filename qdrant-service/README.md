# Qdrant Service - Vector Database for Lost Items

Serwis do zarządzania bazą wektorów dla systemu "Odnalezione Zguby". Zajmuje się przechowywaniem, indeksowaniem i wyszukiwaniem semantycznym rzeczy znalezionych.

## 🎯 Rola w Systemie

Qdrant Service pełni rolę **Service D** w architekturze:

```mermaid
graph TB
    A["Service A<br/>Gateway"] -->|item.submitted| Q1["RabbitMQ<br/>q.lost-items.ingest"]
    Q1 --> B["Service B<br/>CLIP Worker"]
    B -->|item.vectorized| Q2["RabbitMQ<br/>q.lost-items.publish"]
    Q2 --> C["Service C<br/>Publisher"]

    B -->|Upsert Embeddings| D["Service D<br/>Qdrant Vector DB<br/>6333/6334"]
    D -->|Search Similar| B

    style A fill:#00ADD8,stroke:#333,color:#fff
    style B fill:#3776AB,stroke:#333,color:#fff
    style C fill:#00ADD8,stroke:#333,color:#fff
    style D fill:#0071C5,stroke:#333,color:#fff,stroke-width:3px
    style Q1 fill:#ff9900,stroke:#333,color:#fff
    style Q2 fill:#ff9900,stroke:#333,color:#fff
```

## 📊 Cechy

### Qdrant Integration

-   **Create Collection** - Inicjalizacja kolekcji Qdrant z konfiguracją wektorów
-   **Upsert Embeddings** - Insert/update embeddingów z metadanymi
-   **Vector Search** - Wyszukiwanie podobnych przedmiotów (cosine similarity)
-   **Batch Operations** - Efektywne operacje na wielu wektorach
-   **Retrieve by ID** - Pobieranie konkretnego wektora po ID
-   **Delete Vectors** - Usuwanie przedmiotów z bazy
-   **Collection Stats** - Statystyki kolekcji

### Integration Points

-   **Service B (CLIP Worker)** - Publikuje embeddings do wstawienia
-   **Search API** - External search queries
-   **Monitoring** - Queue statistics

### Service Features

-   **Live Reload** - Air support dla development
-   **Statistics** - Periodic queue i service statistics
-   **Graceful Shutdown** - Czyste wyłączanie

## 🚀 Installation

### Prerequisites

```bash
# Go 1.20+
go version

# Docker (for Qdrant DB)
docker --version
```

### Setup

1. Install Go dependencies:

```bash
go mod download
go mod tidy
```

2. Install Air for live reloading (optional):

```bash
go install github.com/cosmtrek/air@latest
```

3. Start infrastructure using Docker Compose (from root):

```bash
docker-compose up -d
```

4. Verify services are running:

```bash
docker-compose ps
```

## 📖 Usage

### Initialize the Handler

```go
handler, err := NewQdrantHandler("localhost:6334", "lost_items")
if err != nil {
    log.Fatal(err)
}
defer handler.Close()
```

### Create a Collection

```go
ctx := context.Background()
// Vector size 384 dla all-MiniLM-L6-v2 / CLIP embeddings
err = handler.CreateCollection(ctx, 384, qdrant.Distance_Cosine)
if err != nil {
    log.Fatal(err)
}
```

### Insert an Embedding

```go
embedding := []float32{...}  // 384-dimensional vector

payload := LostItemPayload{
    ItemID:      "550e8400-e29b-41d4-a716-446655440000",
    Title:       "Znaleziony portfel",
    Description: "Czarny portfel skórzany ze złotą klamrą",
    Category:    "Portfele i torby",
    Location:    "Warszawa, Rynek Starego Miasta",
    DateLost:    time.Now(),
    ImageURL:    "http://minio:9000/lost-items-images/uploads/2024-12-06/uuid.jpg",
    ContactInfo: "biuro@urzad.pl",
}

vectorID, err := handler.UpsertEmbedding(ctx, embedding, payload)
if err != nil {
    log.Fatal(err)
}
```

### Search for Similar Items

```go
queryEmbedding := []float32{...}  // Query embedding

results, err := handler.SearchSimilar(ctx, queryEmbedding, 5, 0.7)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("Title: %s, Score: %.4f, Category: %s\n",
        result.Payload.Title,
        result.Score,
        result.Payload.Category)
}
```

### Batch Insert

```go
embeddings := [][]float32{
    {/* embedding 1 */},
    {/* embedding 2 */},
    {/* embedding 3 */},
}

payloads := []LostItemPayload{
    {ItemID: "id1", Title: "Item 1", ...},
    {ItemID: "id2", Title: "Item 2", ...},
    {ItemID: "id3", Title: "Item 3", ...},
}

ids, err := handler.BatchUpsertEmbeddings(ctx, embeddings, payloads)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Inserted %d items\n", len(ids))
```

### Retrieve by ID

```go
retrieved, err := handler.GetVectorByID(ctx, "550e8400-e29b-41d4-a716-446655440000")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Title: %s\n", retrieved.Payload.Title)
```

### Delete a Vector

```go
err = handler.DeleteVector(ctx, "550e8400-e29b-41d4-a716-446655440000")
if err != nil {
    log.Fatal(err)
}
```

## ▶️ Running the Service

### With Air (Live Reload - Development)

```bash
air
# or
make dev
```

### Direct Execution

```bash
go run .
# or
make run
```

### Build Binary

```bash
go build -o bin/qdrant-service.exe .
# or
make build
```

### Docker (Production)

```bash
docker build -t qdrant-service .
docker run -p 8080:8080 \
  -e RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/ \
  -e QDRANT_ADDR=qdrant-db:6334 \
  -e COLLECTION_NAME=lost_items \
  qdrant-service
```

## ⚙️ Configuration

### Environment Variables

| Variable          | Default                              | Description                                    |
| ----------------- | ------------------------------------ | ---------------------------------------------- |
| `RABBITMQ_URL`    | `amqp://guest:guest@localhost:5672/` | RabbitMQ connection string                     |
| `QDRANT_ADDR`     | `localhost:6334`                     | Qdrant gRPC address                            |
| `COLLECTION_NAME` | `lost_items`                         | Qdrant collection name                         |
| `VECTOR_SIZE`     | `384`                                | Dimension of embeddings                        |
| `DISTANCE_METRIC` | `Cosine`                             | Distance metric (Cosine, Euclidean, Manhattan) |

### Przykład .env

```env
RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/
QDRANT_ADDR=qdrant-db:6334
COLLECTION_NAME=lost_items
```

## 📋 Service Behavior

The service automatically:

1. Connects to RabbitMQ and Qdrant
2. Sets up required message queues
3. Creates Qdrant collection if it doesn't exist
4. Starts consuming messages from:
    - `vector_indexing` queue - Indexes embeddings into Qdrant
    - `embedding_requests` queue - Processes search/embedding requests
5. Displays statistics every 30 seconds

## 📁 Project Structure

```
qdrant-service/
├── main.go                      # Entry point
├── qdrant-handler.go            # Qdrant database operations
├── rabbitmq-handler.go          # RabbitMQ queue operations
├── go.mod
├── go.sum
├── Dockerfile                   # Docker build
├── Makefile                     # Build commands
└── README.md
```

## 🧮 Data Structure

### LostItemPayload

```go
type LostItemPayload struct {
    ItemID      string    // Unique identifier (UUID)
    Title       string    // Item title
    Description string    // Detailed description
    Category    string    // Category (Bags, Electronics, Keys, etc.)
    Location    string    // Where it was lost
    DateLost    time.Time // When it was lost
    ImageURL    string    // Optional image URL from MinIO
    ContactInfo string    // Contact information
    Metadata    map[string]interface{} // Additional metadata
}
```

### Vector Specifications

-   **Dimension**: 384 (compatible with CLIP model)
-   **Distance Metric**: Cosine similarity
-   **Index**: HNSW (Hierarchical Navigable Small World)
-   **Storage**: Persistent (snapshot on disk)

## 📈 Monitoring

### Statistics Display

The service prints statistics every 30 seconds:

```
📊 Service Statistics - 2024-12-06 10:37:00
═════════════════════════════════════════════════════
📦 Qdrant Collection: lost_items
   ├─ Vectors: 1,245
   ├─ Size: 2.3 MB
   └─ Last updated: 2024-12-06 10:35:42

🐇 RabbitMQ Queues:
   ├─ vector_indexing (4 messages, 1 consumer)
   ├─ embedding_requests (2 messages, 1 consumer)
   └─ Total messages: 6

📤 Publish Rate: 12 events/min
📥 Consume Rate: 10 events/min
═════════════════════════════════════════════════════
```

### Health Check

```bash
curl http://localhost:6333/health

# Response
{
  "status": "ok",
  "version": "0.11.0",
  "check": {
    "collections_exist": true,
    "collections_count": 1
  }
}
```

## 🧪 Testing

### Manual Testing with Emulator

```bash
# From event-emulator directory
cd ../event-emulator
make run

# Select: 4. Simulate End-to-End Flow
# Watch Qdrant Service consume and index events
```

### Direct Vector Search Test

```bash
# Query Qdrant directly
curl -X GET http://localhost:6333/dashboard

# Or use gRPC client
grpcurl -plaintext localhost:6334 list
```

### Check Collection

```bash
grpcurl -plaintext localhost:6334 qdrant.Collections/List
```

## 🔧 Make Commands

```bash
make help          # Show available commands
make install-air   # Install Air tool for live reload
make dev           # Run with live reload
make build         # Build binary
make run           # Run directly
make clean         # Clean artifacts
make deps          # Download dependencies
make docker-build  # Build Docker image
make docker-run    # Run Docker container
```

## 🤝 Integration Points

### Service B (CLIP Worker) 🐍

-   Sends: Embeddings to upsert (via `item.vectorized` event or direct gRPC)
-   Receives: Vector IDs and confirmation

### Search Requests

-   External services can query vectors via gRPC
-   Returns top-K similar items with metadata

### Monitoring/Analytics

-   Exposes statistics for system monitoring
-   Collection health checks

## 🐛 Troubleshooting

### Qdrant Connection Failed

```bash
# Check if Qdrant DB is running
docker ps | grep qdrant

# Check logs
docker logs odnalezione-qdrant-db

# Restart if needed
docker-compose restart qdrant-db
```

### Collection Not Created

```bash
# Check collection exists
curl http://localhost:6333/dashboard

# Navigate to Collections tab
# If empty, service will create on first vector insert
```

### Memory Usage High

```bash
# Qdrant uses HNSW which is memory-intensive
# Limit memory in docker-compose.yml

services:
  qdrant-db:
    deploy:
      resources:
        limits:
          memory: 2G
```

### Vector Insert Failed

```bash
# Verify vector dimension matches (should be 384)
# Verify payload is valid JSON
# Check Qdrant logs for errors

docker logs odnalezione-qdrant-service
```

## 📚 Additional Resources

-   [Qdrant Official Docs](https://qdrant.tech/documentation/)
-   [Qdrant Client Libraries](https://qdrant.tech/documentation/concepts/client-libraries/)
-   [Vector Similarity Search](https://qdrant.tech/documentation/concepts/vector-db/)
-   [HNSW Algorithm](https://qdrant.tech/documentation/concepts/indexing/)
-   [CLIP Model Info](https://openai.com/research/learning-transferable-visual-models)

## 🌟 Features

### ✅ Implemented

-   [x] Qdrant collection management
-   [x] Batch upsert operations
-   [x] Semantic search (cosine similarity)
-   [x] RabbitMQ integration
-   [x] Statistics monitoring
-   [x] Graceful shutdown
-   [x] Live reload (development)
-   [x] Docker support

### 🚧 Future Enhancements

-   [ ] Hybrid search (vector + metadata filtering)
-   [ ] Re-ranking capabilities
-   [ ] Approximate nearest neighbor variations
-   [ ] Collection versioning
-   [ ] Backup/restore automation
-   [ ] Performance metrics export
-   [ ] Multi-collection support

## 📄 License

Część projektu Odnalezione Zguby - HackNation

---

**Service D: Qdrant Vector Database** - Baza danych wektorowych dla wyszukiwania semantycznego

## Installation

1. Install Go dependencies:

```powershell
go mod download
```

2. Install Air for live reloading:

```powershell
go install github.com/cosmtrek/air@latest
```

3. Start services using Docker Compose (from root):

```powershell
docker-compose up -d
```

## Usage

### Initialize the Handler

```go
handler, err := NewQdrantHandler("localhost:6334", "lost_items")
if err != nil {
    log.Fatal(err)
}
defer handler.Close()
```

### Create a Collection

```go
ctx := context.Background()
// Vector size 384 for all-MiniLM-L6-v2 embeddings, using cosine distance
err = handler.CreateCollection(ctx, 384, qdrant.Distance_Cosine)
```

### Insert an Embedding

```go
embedding := []float32{...} // Your embedding vector

payload := LostItemPayload{
    Title:       "Lost Blue Backpack",
    Description: "A blue backpack with laptop inside",
    Category:    "Bags",
    Location:    "Central Train Station",
    DateLost:    time.Now(),
    ImageURL:    "https://example.com/image.jpg",
    ContactInfo: "contact@example.com",
}

itemID, err := handler.UpsertEmbedding(ctx, embedding, payload)
```

### Search for Similar Items

```go
queryEmbedding := []float32{...} // Query embedding

results, err := handler.SearchSimilar(ctx, queryEmbedding, 5, 0.7)
// Returns top 5 results with similarity score >= 0.7

for _, result := range results {
    fmt.Printf("Title: %s, Score: %.4f\n", result.Payload.Title, result.Score)
}
```

### Batch Insert

```go
embeddings := [][]float32{
    {/* embedding 1 */},
    {/* embedding 2 */},
    {/* embedding 3 */},
}

payloads := []LostItemPayload{
    {Title: "Item 1", ...},
    {Title: "Item 2", ...},
    {Title: "Item 3", ...},
}

ids, err := handler.BatchUpsertEmbeddings(ctx, embeddings, payloads)
```

### Retrieve by ID

```go
retrieved, err := handler.GetVectorByID(ctx, "item-uuid")
```

### Delete a Vector

```go
err = handler.DeleteVector(ctx, "item-uuid")
```

## Running the Service

### With Air (Live Reload)

```powershell
air
# or
make dev
```

### Direct Execution

```powershell
go run .
# or
make run
```

### Build Binary

```powershell
go build -o bin/qdrant-service.exe .
# or
make build
```

### Run Example

```powershell
go run example_usage.go qdrant-handler.go rabbitmq-handler.go
# or
make example
```

## Service Configuration

Set environment variables:

-   `RABBITMQ_URL` - RabbitMQ connection URL (default: `amqp://guest:guest@localhost:5672/`)
-   `QDRANT_ADDR` - Qdrant gRPC address (default: `localhost:6334`)
-   `COLLECTION_NAME` - Qdrant collection name (default: `lost_items`)

## Service Behavior

The service automatically:

1. Connects to RabbitMQ and Qdrant
2. Sets up required message queues
3. Creates Qdrant collection if it doesn't exist
4. Starts consuming messages from:
    - `vector_indexing` queue - Indexes embeddings into Qdrant
    - `embedding_requests` queue - Processes search/embedding requests
5. Displays statistics every 30 seconds

## Make Commands

```powershell
make help          # Show available commands
make install-air   # Install Air tool
make dev           # Run with live reload
make build         # Build binary
make run           # Run directly
make clean         # Clean artifacts
make deps          # Download dependencies
make example       # Run example usage
```

## Data Structure

### LostItemPayload

```go
type LostItemPayload struct {
    ItemID      string    // Unique identifier
    Title       string    // Item title
    Description string    // Detailed description
    Category    string    // Category (Bags, Electronics, etc.)
    Location    string    // Where it was lost
    DateLost    time.Time // When it was lost
    ImageURL    string    // Optional image URL
    ContactInfo string    // Optional contact information
}
```

### SearchResult

```go
type SearchResult struct {
    ID      string          // Item ID
    Score   float32         // Similarity score (0-1)
    Payload LostItemPayload // Item metadata
}
```

## Configuration

The Qdrant service is configured in `docker-compose.yaml`:

-   gRPC Port: 6334
-   REST API Port: 6333
-   Storage: `./qdrant_storage`

## Distance Metrics

Supported distance metrics:

-   `qdrant.Distance_Cosine` - Cosine similarity (recommended for text embeddings)
-   `qdrant.Distance_Euclidean` - Euclidean distance
-   `qdrant.Distance_Dot` - Dot product
-   `qdrant.Distance_Manhattan` - Manhattan distance

## Example

See `example_usage.go` for a complete working example demonstrating all features.

Run the example:

```bash
go run qdrant-handler.go example_usage.go
```

## Integration with Embedding Models

This handler works with any embedding model that produces fixed-size vectors. Common choices:

-   **all-MiniLM-L6-v2**: 384 dimensions (fast, efficient)
-   **all-mpnet-base-v2**: 768 dimensions (more accurate)
-   **OpenAI text-embedding-ada-002**: 1536 dimensions (high quality)

Make sure to set the correct vector size when creating the collection to match your embedding model.

## Error Handling

All functions return errors that should be checked. Common errors:

-   Connection failures
-   Collection not found
-   Invalid vector dimensions
-   Point not found

## License

See LICENSE file in the root directory.
