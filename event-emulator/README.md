# Event Emulator - RabbitMQ Testing Tool# Event Emulator

Narzędzie do testowania i emulatora zdarzeń dla systemu "Odnalezione Zguby". Umożliwia symulację całego przepływu danych bez potrzeby interfejsu użytkownika.RabbitMQ event emulator for testing the lost items system.

## 📋 Cechy## Features

-   📝 **Item Events** - Emulacja raportów rzeczy znalezionych- 📝 **New Item Events** - Simulate lost item reports

-   🔢 **Vector Indexing** - Symulacja wektoryacji- 🔢 **Vector Index Events** - Generate embeddings for indexing

-   🔍 **Search Requests** - Testowanie wyszukiwania- 🔍 **Search Result Events** - Simulate search matches

-   🔔 **Complete Workflow** - Pełny e2e flow- 🔔 **Notification Events** - Test user notifications

-   💥 **Burst Mode** - Generowanie wielu zdarzeń szybko- 💥 **Burst Mode** - Generate multiple events rapidly

-   🔄 **Continuous Mode** - Ciągłe zdarzenia w losowych intervalach- 🔄 **Continuous Mode** - Generate events at random intervals

-   ⚡ **Stress Testing** - Testy wydajności- 🎬 **End-to-End Flow** - Complete workflow simulation

-   📊 **Queue Statistics** - Monitorowanie kolejek- ⚡ **Stress Testing** - High-volume load testing

-   📊 **Queue Statistics** - Monitor queue status

## 🚀 Installation

## Installation

### Prerequisites

1. Install dependencies:

`bash`powershell

# Go 1.20+go mod download

go version```

# Docker (for RabbitMQ, Qdrant)2. Install Air for live reloading:

docker-compose up -d```powershell

```go install github.com/cosmtrek/air@latest

```

### Setup

## Usage

1. Install dependencies:

```````bash### Run with Air (Live Reload)

go mod download```powershell

go mod tidyair

```# or

make dev

2. Install Air for live reloading (optional):```

```bash

go install github.com/cosmtrek/air@latest### Run directly

``````powershell

go run ./emulator

## ▶️ Usage# or

make run

### Run with Air (Live Reload)```

```bash

air### Build binary

# or```powershell

make devgo build -o bin/emulator.exe ./emulator

```# or

make build

### Run Directly```

```bash

go run ./emulator## Configuration

# or

make runSet environment variables:

```- `RABBITMQ_URL` - RabbitMQ connection URL (default: `amqp://guest:guest@localhost:5672/`)

- `QDRANT_ADDR` - Qdrant address (default: `localhost:6334`)

### Build Binary- `COLLECTION_NAME` - Qdrant collection name (default: `lost_items`)

```bash

go build -o bin/emulator.exe ./emulator## Interactive Menu

# or

make buildWhen you run the emulator, you'll see an interactive menu:

```````

````

## ⚙️ Configuration╔════════════════════════════════════════════════════╗

║       RabbitMQ Event Emulator - Lost Items        ║

Set environment variables:╠════════════════════════════════════════════════════╣

- `RABBITMQ_URL` - RabbitMQ connection URL (default: `amqp://guest:guest@localhost:5672/`)║ 1. Emit Single Event                               ║

- `QDRANT_ADDR` - Qdrant address (default: `localhost:6334`)║ 2. Emit Burst of Events                            ║

- `COLLECTION_NAME` - Qdrant collection name (default: `lost_items`)║ 3. Run Continuous Mode                             ║

║ 4. Simulate End-to-End Flow                        ║

### Example .env║ 5. Stress Test                                     ║

║ 6. Show Queue Statistics                           ║

```env║ 7. Purge All Queues                                ║

RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/║ 8. Emit Custom Message                             ║

QDRANT_ADDR=localhost:6334║                                                    ║

COLLECTION_NAME=lost_items║ h. Show this menu                                  ║

```║ q. Quit                                            ║

╚════════════════════════════════════════════════════╝

## 📱 Interactive Menu```



Kiedy uruchamiasz emulator, zobaczysz menu interaktywne:## Example Usage



```### Single Event

╔════════════════════════════════════════════════════╗Select option 1, then choose event type (1-5)

║       RabbitMQ Event Emulator - Lost Items        ║

╠════════════════════════════════════════════════════╣### Burst Mode

║ 1. Emit Single Event                               ║Select option 2, specify:

║ 2. Emit Burst of Events                            ║- Number of events (e.g., 100)

║ 3. Run Continuous Mode                             ║- Delay between events in ms (e.g., 50)

║ 4. Simulate End-to-End Flow                        ║

║ 5. Stress Test                                     ║### Continuous Mode

║ 6. Show Queue Statistics                           ║Select option 3, specify:

║ 7. Purge All Queues                                ║- Min delay in seconds (e.g., 1)

║ 8. Emit Custom Message                             ║- Max delay in seconds (e.g., 5)

║                                                    ║Press Enter to stop

║ h. Show this menu                                  ║

║ q. Quit                                            ║### Stress Test

╚════════════════════════════════════════════════════╝Select option 5, specify:

- Duration in seconds (e.g., 60)

Select option: _- Events per second (e.g., 100)

````

## Make Commands

## 📚 Usage Examples

````powershell

### 1️⃣ Single Eventmake help          # Show available commands

make install-air   # Install Air tool

```make dev           # Run with live reload

Select option: 1make build         # Build binary

make run           # Run directly

Choose event type:make clean         # Clean artifacts

1. item.submitted (Gateway → Worker)make deps          # Download dependencies

2. item.vectorized (Worker → Publisher)```

3. search_request

4. notification## Sample Data

5. status_update

The emulator includes realistic sample data:

Select (1-5): 1- 15 different item types (backpacks, wallets, phones, keys, etc.)

✅ Emitted: item.submitted event to RabbitMQ- 15 different locations (train station, mall, park, etc.)

```- Various categories (Bags, Electronics, Wallets, Keys, etc.)

- Random embeddings (384 dimensions)

### 2️⃣ Burst Mode

## Queue Management

````

Select option: 2View queue statistics with option 6:

````

Number of events: 100📊 Queue Statistics:

Delay between events (ms): 50═══════════════════════════════════════════════════

embedding_requests      | Messages:   12 | Consumers: 1

✅ Emitting 100 events...vector_indexing         | Messages:    5 | Consumers: 1

[████████████████████] 100%notifications           | Messages:    0 | Consumers: 0

✅ Emitted 100 events in 5.234ssearch_results          | Messages:    3 | Consumers: 1

```═══════════════════════════════════════════════════

````

### 3️⃣ Continuous Mode

## Development

```

Select option: 3The emulator uses Air for live reloading. Any changes to `.go` files will automatically rebuild and restart the application.



Min delay (seconds): 1## License

Max delay (seconds): 5

See LICENSE file in the root directory.

✅ Emitting events continuously...
Press Enter to stop

2025-12-06 10:37:00 - Emitted item.submitted
2025-12-06 10:37:03 - Emitted item.submitted
2025-12-06 10:37:06 - Emitted item.vectorized
...
Enter to stop:
```

### 4️⃣ End-to-End Flow

```
Select option: 4

Simulating complete workflow...

Step 1: Gateway submits item
  ✅ Published item.submitted to q.lost-items.ingest

Step 2: CLIP Worker processes
  ✅ Consumed from q.lost-items.ingest
  ✅ Generated 384-dim embedding
  ✅ Upserted to Qdrant
  ✅ Published item.vectorized to q.lost-items.publish

Step 3: Publisher sends to dane.gov.pl
  ✅ Consumed from q.lost-items.publish
  ✅ Converted to DCAT-AP PL
  ✅ Sent to API

✅ End-to-end flow completed in 4.567s
```

### 5️⃣ Stress Test

```
Select option: 5

Duration (seconds): 60
Events per second: 100

Running stress test for 60s at 100 events/sec...

[████████████████████] 100%
Stats:
  Total events: 6,000
  Duration: 60.234s
  Avg rate: 99.6 events/sec
  Min latency: 2ms
  Max latency: 45ms
  Avg latency: 12.3ms

✅ Stress test completed
```

### 6️⃣ Queue Statistics

```
Select option: 6

📊 Queue Statistics:
═════════════════════════════════════════════════════

Exchange: lost-found.events (topic)

Queue: q.lost-items.ingest
  ├─ Messages: 12
  ├─ Consumers: 1 (Service B)
  ├─ Ready: 12
  └─ Unacked: 0

Queue: q.lost-items.publish
  ├─ Messages: 5
  ├─ Consumers: 1 (Service C)
  ├─ Ready: 5
  └─ Unacked: 0

═════════════════════════════════════════════════════
Total messages in system: 17
Last 5 min rate: 45 events/min
```

### 7️⃣ Purge All Queues

```
Select option: 7

⚠️  This will delete all messages!
Continue? (y/n): y

✅ Purging q.lost-items.ingest... Done
✅ Purging q.lost-items.publish... Done
✅ All queues purged
```

### 8️⃣ Custom Message

```
Select option: 8

Exchange: lost-found.events
Routing Key: item.submitted
Message (JSON): {"id":"test-123","title":"Test Item"}

✅ Published custom message
```

## 📊 Sample Data

Emulator zawiera realistyczne dane testowe:

-   **15+ typów przedmiotów:** portfele, telefony, klucze, plecaki, okulary, itd.
-   **15+ lokalizacji:** dworzec, park, centrum handlowe, metr, plaża, itp.
-   **Różne kategorie:** Elektronika, Portfele, Klucze, Akcesoria, Ubrania, itp.
-   **Losowe embedingi:** 384-wymiarowe wektory CLIP
-   **Polskie teksty:** Realistyczne opisy przedmiotów

## 🔧 Make Commands

```bash
make help          # Show available commands
make install-air   # Install Air tool
make dev           # Run with live reload
make build         # Build binary
make run           # Run directly
make clean         # Clean artifacts
make deps          # Download dependencies
```

## 📈 Monitoring During Tests

### Terminal 1: Monitor RabbitMQ

```bash
# Watch queue stats in real-time
watch -n 1 'curl -s http://localhost:15672/api/queues admin:admin123 | jq'

# Or open UI
# http://localhost:15672 (admin/admin123)
```

### Terminal 2: Monitor Qdrant

```bash
# Check collection
curl http://localhost:6333/collections/lost_items

# Or open dashboard
# http://localhost:6333/dashboard
```

### Terminal 3: Monitor Logs

```bash
# From service-a-gateway
go run cmd/server/main.go

# From qdrant-service
go run .
```

## 🐛 Troubleshooting

### RabbitMQ Not Running

```bash
docker-compose ps rabbitmq

# Start if needed
docker-compose up -d rabbitmq
```

### Connection Refused

```bash
# Check RABBITMQ_URL
echo $RABBITMQ_URL

# Should be: amqp://admin:admin123@rabbitmq:5672/
# For local testing: amqp://admin:admin123@localhost:5672/
```

### No Messages in Queue

```bash
# 1. Check if emulator is running
# 2. Check if consumers are connected
# 3. Check RabbitMQ logs

docker logs odnalezione-rabbitmq

# 4. Try purging and re-emitting
```

### Stress Test Failures

```bash
# Reduce events per second
# If network is slow, adjust rate down

# Or increase RabbitMQ memory
docker-compose.yml:
  rabbitmq:
    deploy:
      resources:
        limits:
          memory: 1G
```

## 🎯 Testing Scenarios

### Scenario 1: Basic Functionality

```
1. Start emulator
2. Option 1: Emit single item.submitted
3. Option 6: Check queue stats
4. Verify message in q.lost-items.ingest
```

### Scenario 2: End-to-End

```
1. Start all services
2. Option 4: Simulate End-to-End Flow
3. Watch as data flows through system
4. Verify in Qdrant and dane.gov.pl
```

### Scenario 3: Load Testing

```
1. Option 5: Stress Test
2. Duration: 300 seconds
3. Rate: 50 events/sec
4. Monitor RabbitMQ and services
```

### Scenario 4: Recovery

```
1. Option 2: Burst 1000 events
2. Kill CLIP Worker (Ctrl+C)
3. Messages should queue
4. Restart CLIP Worker
5. Messages should be consumed
```

## 📄 Event Schemas

### item.submitted

```json
{
    "id": "uuid",
    "title": "string",
    "description": "string",
    "category": "string",
    "location": "string",
    "found_date": "ISO8601",
    "image_url": "string",
    "contact_info": "string",
    "timestamp": "ISO8601"
}
```

### item.vectorized

```json
{
  "id": "uuid",
  "request_id": "uuid",
  "original_data": {...},
  "vector_embedding": [0.123, ...],
  "vector_id": "string",
  "embedding_model": "CLIP",
  "embedding_dimension": 384,
  "processing_time_ms": 1234,
  "processed_at": "ISO8601"
}
```

## 🌟 Features

### ✅ Implemented

-   [x] Interactive menu
-   [x] Single event emission
-   [x] Burst mode
-   [x] Continuous emission
-   [x] End-to-end simulation
-   [x] Stress testing
-   [x] Queue monitoring
-   [x] Queue purge
-   [x] Custom messages
-   [x] Realistic sample data
-   [x] Live reload

### 🚧 Future Enhancements

-   [ ] Event replay from saved files
-   [ ] Performance profiling
-   [ ] Dead letter queue handling
-   [ ] Consumer simulation
-   [ ] Failure injection
-   [ ] Latency simulation
-   [ ] Visual dashboard

## 📚 Additional Resources

-   [RabbitMQ Tutorials](https://www.rabbitmq.com/getstarted.html)
-   [AMQP Concepts](https://www.rabbitmq.com/tutorials/amqp-concepts.html)
-   [Testing Message Queues](https://www.rabbitmq.com/testing.html)

## 📄 License

Część projektu Odnalezione Zguby - HackNation

---

**Event Emulator** - Testing tool for RabbitMQ message flows
