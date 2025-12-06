FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git gcc musl-dev
WORKDIR /app

COPY go.mod ./
COPY go.sum* ./

RUN go mod download && go mod verify
COPY main.go qdrant-handler.go rabbitmq-handler.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o qdrant-service .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app
COPY --from=builder /app/qdrant-service .
RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8080

CMD ["./qdrant-service"]
