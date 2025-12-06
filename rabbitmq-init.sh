#!/bin/bash
set -e

# Configuration
RABBITMQ_HOST=${RABBITMQ_HOST:-"rabbitmq"}
RABBITMQ_USER=${RABBITMQ_USER:-"admin"}
RABBITMQ_PASS=${RABBITMQ_PASS:-"admin123"}
API_URL="http://${RABBITMQ_HOST}:15672/api"

echo "⏳ Waiting for RabbitMQ management API at ${API_URL}..."

# Wait for RabbitMQ to be ready
for i in {1..60}; do
  if curl -s -u "${RABBITMQ_USER}:${RABBITMQ_PASS}" "${API_URL}/aliveness-test/%2F" > /dev/null 2>&1; then
    echo "✅ RabbitMQ is ready!"
    break
  fi
  echo "zzz... Attempt $i/60: Waiting for RabbitMQ..."
  sleep 2
done

# Function to create resource
create_resource() {
    local method=$1
    local path=$2
    local data=$3
    local description=$4

    echo "Resource: $description"
    response=$(curl -s -w "%{http_code}" -o /dev/null -u "${RABBITMQ_USER}:${RABBITMQ_PASS}" \
        -H "content-type:application/json" \
        -X "${method}" "${API_URL}${path}" \
        -d "${data}")

    if [[ "$response" =~ ^2 ]]; then
        echo "   ✅ Created/Updated (HTTP $response)"
    else
        echo "   ❌ Failed (HTTP $response)"
        # exit 1 # Optional: exit on failure? For now, we continue but log error.
    fi
}

echo "---------------------------------------------------"
echo "🐰 Configuring RabbitMQ Exchanges, Queues, and Bindings"
echo "---------------------------------------------------"

# Create Exchange
create_resource "PUT" "/exchanges/%2F/lost-found.events" \
    '{"type":"topic","durable":true}' \
    "Exchange 'lost-found.events'"

# Create Queues
create_resource "PUT" "/queues/%2F/q.lost-items.ingest" \
    '{"durable":true,"arguments":{}}' \
    "Queue 'q.lost-items.ingest'"

create_resource "PUT" "/queues/%2F/q.lost-items.publish" \
    '{"durable":true,"arguments":{}}' \
    "Queue 'q.lost-items.publish'"

# Create Bindings
create_resource "POST" "/bindings/%2F/e/lost-found.events/q/q.lost-items.ingest" \
    '{"routing_key":"item.submitted"}' \
    "Binding 'q.lost-items.ingest' -> 'item.submitted'"

create_resource "POST" "/bindings/%2F/e/lost-found.events/q/q.lost-items.publish" \
    '{"routing_key":"item.vectorized"}' \
    "Binding 'q.lost-items.publish' -> 'item.vectorized'"

echo "---------------------------------------------------"
echo "✨ RabbitMQ init complete!"
