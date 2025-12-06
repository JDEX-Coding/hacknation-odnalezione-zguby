# Qdrant Service - Lost Items Vector Search

This service handles embedding storage and retrieval for the lost items database using Qdrant vector database and RabbitMQ message queue.

## Features

### Qdrant Integration
- **Create Collection**: Initialize a Qdrant collection with custom vector size and distance metric
- **Upsert Embeddings**: Insert or update item embeddings with associated metadata
- **Vector Search**: Find similar items using cosine similarity or other distance metrics
- **Batch Operations**: Efficiently insert multiple embeddings at once
- **Retrieve by ID**: Get specific vectors and metadata by item ID
- **Delete Vectors**: Remove items from the database

### RabbitMQ Integration
- **Message Consumers**: Process vector indexing and search requests
- **Queue Management**: Monitor and manage message queues
- **Auto-acknowledgment**: Reliable message processing with acknowledgments

### Service Features
- **Live Reload**: Air support for hot-reloading during development
- **Statistics**: Periodic queue and service statistics
- **Graceful Shutdown**: Clean shutdown handling

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
- `RABBITMQ_URL` - RabbitMQ connection URL (default: `amqp://guest:guest@localhost:5672/`)
- `QDRANT_ADDR` - Qdrant gRPC address (default: `localhost:6334`)
- `COLLECTION_NAME` - Qdrant collection name (default: `lost_items`)

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
- gRPC Port: 6334
- REST API Port: 6333
- Storage: `./qdrant_storage`

## Distance Metrics

Supported distance metrics:
- `qdrant.Distance_Cosine` - Cosine similarity (recommended for text embeddings)
- `qdrant.Distance_Euclidean` - Euclidean distance
- `qdrant.Distance_Dot` - Dot product
- `qdrant.Distance_Manhattan` - Manhattan distance

## Example

See `example_usage.go` for a complete working example demonstrating all features.

Run the example:
```bash
go run qdrant-handler.go example_usage.go
```

## Integration with Embedding Models

This handler works with any embedding model that produces fixed-size vectors. Common choices:

- **all-MiniLM-L6-v2**: 384 dimensions (fast, efficient)
- **all-mpnet-base-v2**: 768 dimensions (more accurate)
- **OpenAI text-embedding-ada-002**: 1536 dimensions (high quality)

Make sure to set the correct vector size when creating the collection to match your embedding model.

## Error Handling

All functions return errors that should be checked. Common errors:
- Connection failures
- Collection not found
- Invalid vector dimensions
- Point not found

## License

See LICENSE file in the root directory.
