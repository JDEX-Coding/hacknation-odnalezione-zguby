# Event Emulator

RabbitMQ event emulator for testing the lost items system.

## Features

- 📝 **New Item Events** - Simulate lost item reports
- 🔢 **Vector Index Events** - Generate embeddings for indexing
- 🔍 **Search Result Events** - Simulate search matches
- 🔔 **Notification Events** - Test user notifications
- 💥 **Burst Mode** - Generate multiple events rapidly
- 🔄 **Continuous Mode** - Generate events at random intervals
- 🎬 **End-to-End Flow** - Complete workflow simulation
- ⚡ **Stress Testing** - High-volume load testing
- 📊 **Queue Statistics** - Monitor queue status

## Installation

1. Install dependencies:
```powershell
go mod download
```

2. Install Air for live reloading:
```powershell
go install github.com/cosmtrek/air@latest
```

## Usage

### Run with Air (Live Reload)
```powershell
air
# or
make dev
```

### Run directly
```powershell
go run ./emulator
# or
make run
```

### Build binary
```powershell
go build -o bin/emulator.exe ./emulator
# or
make build
```

## Configuration

Set environment variables:
- `RABBITMQ_URL` - RabbitMQ connection URL (default: `amqp://guest:guest@localhost:5672/`)
- `QDRANT_ADDR` - Qdrant address (default: `localhost:6334`)
- `COLLECTION_NAME` - Qdrant collection name (default: `lost_items`)

## Interactive Menu

When you run the emulator, you'll see an interactive menu:

```
╔════════════════════════════════════════════════════╗
║       RabbitMQ Event Emulator - Lost Items        ║
╠════════════════════════════════════════════════════╣
║ 1. Emit Single Event                               ║
║ 2. Emit Burst of Events                            ║
║ 3. Run Continuous Mode                             ║
║ 4. Simulate End-to-End Flow                        ║
║ 5. Stress Test                                     ║
║ 6. Show Queue Statistics                           ║
║ 7. Purge All Queues                                ║
║ 8. Emit Custom Message                             ║
║                                                    ║
║ h. Show this menu                                  ║
║ q. Quit                                            ║
╚════════════════════════════════════════════════════╝
```

## Example Usage

### Single Event
Select option 1, then choose event type (1-5)

### Burst Mode
Select option 2, specify:
- Number of events (e.g., 100)
- Delay between events in ms (e.g., 50)

### Continuous Mode
Select option 3, specify:
- Min delay in seconds (e.g., 1)
- Max delay in seconds (e.g., 5)
Press Enter to stop

### Stress Test
Select option 5, specify:
- Duration in seconds (e.g., 60)
- Events per second (e.g., 100)

## Make Commands

```powershell
make help          # Show available commands
make install-air   # Install Air tool
make dev           # Run with live reload
make build         # Build binary
make run           # Run directly
make clean         # Clean artifacts
make deps          # Download dependencies
```

## Sample Data

The emulator includes realistic sample data:
- 15 different item types (backpacks, wallets, phones, keys, etc.)
- 15 different locations (train station, mall, park, etc.)
- Various categories (Bags, Electronics, Wallets, Keys, etc.)
- Random embeddings (384 dimensions)

## Queue Management

View queue statistics with option 6:
```
📊 Queue Statistics:
═══════════════════════════════════════════════════
embedding_requests      | Messages:   12 | Consumers: 1
vector_indexing         | Messages:    5 | Consumers: 1
notifications           | Messages:    0 | Consumers: 0
search_results          | Messages:    3 | Consumers: 1
═══════════════════════════════════════════════════
```

## Development

The emulator uses Air for live reloading. Any changes to `.go` files will automatically rebuild and restart the application.

## License

See LICENSE file in the root directory.
