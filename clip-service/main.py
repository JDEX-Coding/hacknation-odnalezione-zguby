#!/usr/bin/env python3
"""
CLIP Service - Image and Text Embedding Service
Consumes messages from RabbitMQ, downloads images, generates embeddings using CLIP,
and publishes results back to RabbitMQ for indexing in Qdrant.
"""

import os
import sys
import json
import logging
import time
from typing import Dict, Any, Optional
from datetime import datetime
from pathlib import Path

import pika
from pika.exceptions import AMQPConnectionError, ChannelClosedByBroker
from dotenv import load_dotenv

from clip_handler import CLIPHandler
from minio_handler import MinIOHandler
from message_converter import MessageConverter

# Load environment variables from .env file
load_dotenv()

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Configuration from environment variables
RABBITMQ_URL = os.getenv('RABBITMQ_URL', 'amqp://admin:admin123@localhost:5672/')

# RabbitMQ constants
EXCHANGE_NAME = 'lost-found.events'
ROUTING_KEY_SUBMITTED = 'item.submitted'  # Input: from Gateway
ROUTING_KEY_EMBEDDED = 'item.embedded'    # Output: to Qdrant Service
QUEUE_EMBED = 'q.lost-items.embed'        # Input queue: consume from here
QUEUE_INGEST = 'q.lost-items.ingest'      # Output queue: publish to here


class CLIPService:
    """Main service that processes lost items messages."""
    
    def __init__(self):
        """Initialize the CLIP service."""
        self.clip_handler = CLIPHandler()
        self.minio_handler = MinIOHandler()
        self.message_converter = MessageConverter()
        self.connection: Optional[pika.BlockingConnection] = None
        self.channel: Optional[pika.channel.Channel] = None
        
    def connect_rabbitmq(self, max_retries: int = 5, retry_delay: int = 5):
        """Connect to RabbitMQ with retry logic."""
        for attempt in range(max_retries):
            try:
                logger.info(f"Connecting to RabbitMQ (attempt {attempt + 1}/{max_retries})...")
                parameters = pika.URLParameters(RABBITMQ_URL)
                parameters.heartbeat = 600
                parameters.blocked_connection_timeout = 300
                
                self.connection = pika.BlockingConnection(parameters)
                self.channel = self.connection.channel()
                
                # Declare exchange
                self.channel.exchange_declare(
                    exchange=EXCHANGE_NAME,
                    exchange_type='topic',
                    durable=True
                )
                
                # Declare queues
                self.channel.queue_declare(queue=QUEUE_EMBED, durable=True)
                self.channel.queue_declare(queue=QUEUE_INGEST, durable=True)
                
                # Bind input queue to exchange
                self.channel.queue_bind(
                    exchange=EXCHANGE_NAME,
                    queue=QUEUE_EMBED,
                    routing_key=ROUTING_KEY_SUBMITTED
                )
                
                # Set QoS to process one message at a time
                self.channel.basic_qos(prefetch_count=1)
                
                logger.info("Successfully connected to RabbitMQ")
                return
                
            except AMQPConnectionError as e:
                logger.error(f"Failed to connect to RabbitMQ: {e}")
                if attempt < max_retries - 1:
                    logger.info(f"Retrying in {retry_delay} seconds...")
                    time.sleep(retry_delay)
                else:
                    logger.error("Max retries reached. Exiting.")
                    sys.exit(1)
    
    def process_message(
        self,
        ch: pika.channel.Channel,
        method: pika.spec.Basic.Deliver,
        properties: pika.spec.BasicProperties,
        body: bytes
    ):
        """Process incoming message from RabbitMQ."""
        try:
            # Parse message
            raw_message = json.loads(body.decode('utf-8'))
            logger.info(f"Received message: {raw_message.get('item_id', 'unknown')}")
            
            # Normalize message format (handles both image_url and image_key)
            message = self.message_converter.normalize_message(raw_message)
            
            # Validate message
            is_valid, error_msg = self.message_converter.validate_message(message)
            if not is_valid:
                logger.error(f"Invalid message: {error_msg}")
                ch.basic_ack(delivery_tag=method.delivery_tag)
                return
            
            # Extract data from normalized message
            item_id = message.get('item_id')
            text = message.get('text', '')
            description = message.get('description', '')
            category = message.get('category', '')
            image_key = message.get('image_key', '')  # MinIO object key/name
            location = message.get('location', '')
            date_lost = message.get('date_lost', '')
            contact_info = message.get('contact_info', '')
            
            # Combine text for embedding
            combined_text = f"{text}. {description}. Category: {category}"
            
            # Generate text embedding
            logger.info(f"Generating text embedding for: {text}")
            text_embedding = self.clip_handler.encode_text(combined_text)
            
            # Process image if MinIO key is provided
            image_embedding = None
            if image_key:
                try:
                    logger.info(f"Downloading image from MinIO: {image_key}")
                    image_path = self.minio_handler.download_image(image_key, item_id)
                    
                    if image_path:
                        logger.info(f"Generating image embedding")
                        image_embedding = self.clip_handler.encode_image(image_path)
                        
                        # Clean up downloaded image
                        self.minio_handler.cleanup_image(image_path)
                except Exception as e:
                    logger.error(f"Error processing image: {e}")
            
            # Combine embeddings (average if both exist)
            if image_embedding is not None:
                import numpy as np
                final_embedding = ((text_embedding + image_embedding) / 2).tolist()
                logger.info("Combined text and image embeddings")
            else:
                final_embedding = text_embedding.tolist()
                logger.info("Using text embedding only")
            
            # Create standardized output message
            logger.info(f"Publishing vectorized message with embedding")
            vectorized_message = self.message_converter.create_output_message(
                input_message=message,
                embedding=final_embedding,
                has_image_embedding=image_embedding is not None,
                timestamp=datetime.utcnow().isoformat()
            )
            
            self.channel.basic_publish(
                exchange=EXCHANGE_NAME,
                routing_key=ROUTING_KEY_EMBEDDED,
                body=json.dumps(vectorized_message).encode('utf-8'),
                properties=pika.BasicProperties(
                    content_type='application/json',
                    delivery_mode=2  # Persistent
                )
            )
            
            logger.info(f"Successfully processed item: {item_id}")
            ch.basic_ack(delivery_tag=method.delivery_tag)
            
        except json.JSONDecodeError as e:
            logger.error(f"Failed to decode message: {e}")
            ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)
        except Exception as e:
            logger.error(f"Error processing message: {e}", exc_info=True)
            # Requeue message for retry
            ch.basic_nack(delivery_tag=method.delivery_tag, requeue=True)
    
    def start(self):
        """Start consuming messages."""
        logger.info("Starting CLIP Service...")
        
        # Initialize CLIP model
        logger.info("Loading CLIP model...")
        self.clip_handler.load_model()
        
        # Connect to RabbitMQ
        self.connect_rabbitmq()
        
        # Start consuming
        logger.info(f"Listening on queue: {QUEUE_EMBED}")
        logger.info("Waiting for messages. Press CTRL+C to exit.")
        
        self.channel.basic_consume(
            queue=QUEUE_EMBED,
            on_message_callback=self.process_message
        )
        
        try:
            self.channel.start_consuming()
        except KeyboardInterrupt:
            logger.info("Stopping service...")
            self.channel.stop_consuming()
        finally:
            if self.connection and not self.connection.is_closed:
                self.connection.close()
            logger.info("Service stopped")


def main():
    """Main entry point."""
    service = CLIPService()
    service.start()


if __name__ == '__main__':
    main()
