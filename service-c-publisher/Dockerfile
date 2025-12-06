FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application (only main.go, not mock-api.go)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o publisher main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/publisher .

# Run the application
CMD ["./publisher"]
