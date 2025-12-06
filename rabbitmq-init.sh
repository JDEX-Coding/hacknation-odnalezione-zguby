#!/bin/bash

# Wait for RabbitMQ to be ready
sleep 10

# Create the topic exchange
rabbitmqadmin -u admin -p admin123 declare exchange name=lost-found.events type=topic durable=true

# Create queues
rabbitmqadmin -u admin -p admin123 declare queue name=q.lost-items.embed durable=true
rabbitmqadmin -u admin -p admin123 declare queue name=q.lost-items.ingest durable=true
rabbitmqadmin -u admin -p admin123 declare queue name=q.lost-items.publish durable=true

# Bind queues to exchange with routing keys
rabbitmqadmin -u admin -p admin123 declare binding source=lost-found.events destination=q.lost-items.embed routing_key=item.submitted
rabbitmqadmin -u admin -p admin123 declare binding source=lost-found.events destination=q.lost-items.ingest routing_key=item.embedded
rabbitmqadmin -u admin -p admin123 declare binding source=lost-found.events destination=q.lost-items.publish routing_key=item.vectorized

echo "RabbitMQ exchange and queues configured successfully!"
