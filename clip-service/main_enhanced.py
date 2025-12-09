#!/usr/bin/env python3
"""
CLIP Service - Enhanced Image and Text Embedding Service
NOW WITH UNIFIED EMBEDDINGS FOR BETTER SEARCH ACCURACY!

Key improvements:
- Uses SINGLE CLIP model for both text and images (same embedding space)
- Enhanced text preprocessing for Polish language
- Query enhancement with translations
- Proper embedding combination with weights
"""

import os
import sys
import json
import logging
import time
import threading
from typing import Dict, Any, Optional
from datetime import datetime
from pathlib import Path

import pika
from pika.exceptions import AMQPConnectionError, ChannelClosedByBroker
from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import uvicorn

from clip_handler_enhanced import CLIPHandler, TextPreprocessor, QueryEnhancer
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
API_PORT = int(os.getenv('API_PORT', 8000))

# RabbitMQ constants
EXCHANGE_NAME = 'lost-found.events'
ROUTING_KEY_SUBMITTED = 'item.submitted'  # Input: from Gateway
ROUTING_KEY_EMBEDDED = 'item.embedded'    # Output: to Qdrant Service
QUEUE_EMBED = 'q.lost-items.embed'        # Input queue: consume from here
QUEUE_INGEST = 'q.lost-items.ingest'      # Output queue: publish to here


class CLIPService:
    """Main service that processes lost items messages with ENHANCED embeddings."""

    def __init__(self):
        """Initialize the CLIP service with enhanced components."""
        logger.info("🚀 Initializing ENHANCED CLIP Service...")
        self.clip_handler = CLIPHandler()  # Now uses unified model!
        self.minio_handler = MinIOHandler()
        self.message_converter = MessageConverter()
        self.text_preprocessor = TextPreprocessor()
        self.query_enhancer = QueryEnhancer()
        self.connection: Optional[pika.BlockingConnection] = None
        self.channel: Optional[pika.channel.Channel] = None

        # Statistics
        self.stats = {
            'processed': 0,
            'with_images': 0,
            'text_only': 0,
            'errors': 0,
        }

    def initialize(self):
        """Initialize model and resources."""
        logger.info("Loading unified CLIP model...")
        self.clip_handler.load_model()

        # Log configuration
        embedding_dim = self.clip_handler.get_embedding_dimension()
        logger.info(f"✅ Embedding dimension: {embedding_dim}")
        logger.info(f"✅ Using UNIFIED model for text and images")
        logger.info(f"✅ Enhanced preprocessing enabled")

    def connect_rabbitmq(self):
        """Connect to RabbitMQ."""
        try:
            logger.info(f"Connecting to RabbitMQ: {RABBITMQ_URL}")
            self.connection = pika.BlockingConnection(pika.URLParameters(RABBITMQ_URL))
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

            # Bind queue
            self.channel.queue_bind(
                exchange=EXCHANGE_NAME,
                queue=QUEUE_EMBED,
                routing_key=ROUTING_KEY_SUBMITTED
            )

            # Set QoS to process one message at a time (embedding is CPU/GPU intensive)
            self.channel.basic_qos(prefetch_count=1)

            logger.info("✅ Connected to RabbitMQ")

        except Exception as e:
            logger.error(f"Failed to connect to RabbitMQ: {e}")
            raise

    def process_message(
        self,
        ch: pika.channel.Channel,
        method: pika.spec.Basic.Deliver,
        properties: pika.spec.BasicProperties,
        body: bytes
    ):
        """Process incoming message from RabbitMQ with ENHANCED embeddings."""
        try:
            raw_message = json.loads(body.decode('utf-8'))
            logger.info(f"📨 Processing message: {raw_message.get('id', 'unknown')}")

            # Convert message to normalized format
            message = self.message_converter.convert_to_normalized(raw_message)

            if not message:
                logger.error("Failed to normalize message")
                ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)
                return

            # Extract data from normalized message
            item_id = message.get('item_id')
            title = message.get('title', '')
            description = message.get('description', '')
            category = message.get('category', '')
            location = message.get('location', '')
            image_key = message.get('image_key', '')  # MinIO object key/name

            # ENHANCED: Create optimized text for embedding
            logger.info(f"🔧 Creating enhanced text representation")
            enhanced_text = self.text_preprocessor.create_enhanced_text(
                title=title,
                description=description,
                category=category,
                location=location
            )
            logger.info(f"Enhanced text: {enhanced_text[:100]}...")

            # Generate text embedding with unified model
            logger.info(f"📝 Generating text embedding")
            text_embedding = self.clip_handler.encode_text(enhanced_text)
            logger.info(f"✅ Text embedding: shape={text_embedding.shape}, norm={np.linalg.norm(text_embedding):.4f}")

            # Process image if MinIO key is provided
            image_embedding = None
            if image_key:
                try:
                    logger.info(f"🖼️ Downloading image from MinIO: {image_key}")
                    image_path = self.minio_handler.download_image(image_key, item_id)

                    if image_path:
                        logger.info(f"🎨 Generating image embedding")
                        image_embedding = self.clip_handler.encode_image(image_path)
                        logger.info(f"✅ Image embedding: shape={image_embedding.shape}, norm={np.linalg.norm(image_embedding):.4f}")

                        # Calculate text-image similarity (for monitoring)
                        similarity = self.clip_handler.compute_similarity(text_embedding, image_embedding)
                        logger.info(f"📊 Text-Image similarity: {similarity:.4f}")

                        # Clean up downloaded image
                        self.minio_handler.cleanup_image(image_path)
                        self.stats['with_images'] += 1
                except Exception as e:
                    logger.error(f"Error processing image: {e}")

            # ENHANCED: Combine embeddings with optimal weighting
            # Text slightly more weighted (60%) as it's more reliable
            if image_embedding is not None:
                logger.info("🔀 Combining text and image embeddings (60% text, 40% image)")
                final_embedding = self.clip_handler.combine_embeddings(
                    text_embedding=text_embedding,
                    image_embedding=image_embedding,
                    text_weight=0.6  # Slightly favor text
                )
                logger.info("✅ Combined embeddings in unified semantic space")
            else:
                final_embedding = text_embedding
                logger.info("✅ Using text embedding only")
                self.stats['text_only'] += 1

            # Verify embedding dimension
            if len(final_embedding) != 512:
                logger.error(f"❌ Wrong embedding dimension: {len(final_embedding)}, expected 512")
                raise ValueError(f"Embedding dimension mismatch: {len(final_embedding)} != 512")

            # Create standardized output message
            logger.info(f"📤 Publishing vectorized message with 512D embedding")
            vectorized_message = self.message_converter.create_output_message(
                input_message=message,
                embedding=final_embedding.tolist(),
                has_image_embedding=image_embedding is not None,
                timestamp=datetime.utcnow().isoformat() + "Z"
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

            self.stats['processed'] += 1
            logger.info(f"✅ Successfully processed item: {item_id}")
            logger.info(f"📊 Stats: {self.stats}")
            ch.basic_ack(delivery_tag=method.delivery_tag)

        except json.JSONDecodeError as e:
            logger.error(f"Failed to decode message: {e}")
            self.stats['errors'] += 1
            ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)
        except Exception as e:
            logger.error(f"Error processing message: {e}", exc_info=True)
            self.stats['errors'] += 1
            # Requeue message for retry
            ch.basic_nack(delivery_tag=method.delivery_tag, requeue=True)

    def start_consumer_loop(self):
        """Start consuming messages in a loop."""
        logger.info("🎧 Starting RabbitMQ Consumer Loop...")
        self.connect_rabbitmq()

        if not self.channel:
            logger.error("Could not connect to RabbitMQ for consumer loop.")
            return

        logger.info(f"👂 Listening on queue: {QUEUE_EMBED}")

        try:
            self.channel.basic_consume(
                queue=QUEUE_EMBED,
                on_message_callback=self.process_message
            )

            logger.info("✅ Consumer started, waiting for messages...")
            self.channel.start_consuming()

        except KeyboardInterrupt:
            logger.info("Consumer stopped by user")
            self.channel.stop_consuming()
        except Exception as e:
            logger.error(f"Consumer error: {e}")
        finally:
            if self.connection:
                self.connection.close()


# Add numpy import for logging
import numpy as np

# FastAPI App
app = FastAPI(title="CLIP Service Enhanced", version="2.0.0")
service_instance: Optional[CLIPService] = None


class EmbedRequest(BaseModel):
    text: str


class SearchRequest(BaseModel):
    """Enhanced search request with query optimization."""
    query: str
    enhance: bool = True  # Whether to enhance query


@app.on_event("startup")
async def startup_event():
    global service_instance
    logger.info("🚀 Starting ENHANCED CLIP Service API...")
    service_instance = CLIPService()
    service_instance.initialize()

    # Start consumer in background thread
    consumer_thread = threading.Thread(target=service_instance.start_consumer_loop, daemon=True)
    consumer_thread.start()
    logger.info("✅ API ready!")


@app.post("/embed")
async def embed_text(request: EmbedRequest):
    """Generate embedding for text (for search queries)."""
    if not service_instance:
        raise HTTPException(status_code=503, detail="Service not initialized")

    try:
        # Optionally enhance the query
        text = request.text

        embedding = service_instance.clip_handler.encode_text(text)

        return {
            "embedding": embedding.tolist(),
            "dimension": len(embedding),
            "model": "openai/clip-vit-base-patch32",
            "text": text
        }
    except Exception as e:
        logger.error(f"Embedding error: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/search/embed")
async def embed_search_query(request: SearchRequest):
    """
    Generate embedding for search query with optional enhancement.
    This endpoint is optimized for search queries.
    """
    if not service_instance:
        raise HTTPException(status_code=503, detail="Service not initialized")

    try:
        query = request.query
        original_query = query

        # Enhance query if requested
        if request.enhance:
            query = service_instance.query_enhancer.enhance_query(query)
            logger.info(f"Enhanced query: '{original_query}' -> '{query}'")

        embedding = service_instance.clip_handler.encode_text(query)

        return {
            "embedding": embedding.tolist(),
            "dimension": len(embedding),
            "model": "openai/clip-vit-base-patch32",
            "original_query": original_query,
            "enhanced_query": query if request.enhance else None
        }
    except Exception as e:
        logger.error(f"Search embedding error: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    if not service_instance:
        return {"status": "initializing"}

    return {
        "status": "healthy",
        "model": "openai/clip-vit-base-patch32",
        "embedding_dimension": 512,
        "stats": service_instance.stats,
        "features": {
            "unified_embeddings": True,
            "enhanced_preprocessing": True,
            "query_enhancement": True,
            "proper_normalization": True
        }
    }


@app.get("/stats")
async def get_stats():
    """Get service statistics."""
    if not service_instance:
        return {"error": "Service not initialized"}

    return service_instance.stats


def main():
    """Main entry point."""
    logger.info("=" * 80)
    logger.info("🚀 CLIP Service Enhanced - v2.0")
    logger.info("=" * 80)
    logger.info("✨ Features:")
    logger.info("  - Unified CLIP model for text & images")
    logger.info("  - 512-dimensional embeddings")
    logger.info("  - Enhanced text preprocessing")
    logger.info("  - Query enhancement with translations")
    logger.info("  - Proper embedding combination")
    logger.info("=" * 80)

    uvicorn.run(app, host="0.0.0.0", port=API_PORT)


if __name__ == "__main__":
    main()
