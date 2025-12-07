"""
Main service for legacy data conversion.
Consumes files from RabbitMQ queue and publishes converted items.
"""

import os
import json
import uuid
import tempfile
import base64
from datetime import datetime
from pathlib import Path
from typing import List, Dict, Any, Optional
import logging

from text_extractor import TextExtractor
from nlp_converter import NLPConverter
from rabbitmq_handler import RabbitMQHandler
from minio_handler import MinIOHandler

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configuration
RABBITMQ_URL = os.getenv('RABBITMQ_URL', 'amqp://admin:admin123@rabbitmq:5672/')
RABBITMQ_EXCHANGE = os.getenv('RABBITMQ_EXCHANGE', 'lost-found.events')
MINIO_ENDPOINT = os.getenv('MINIO_ENDPOINT', 'minio:9000')
MINIO_ACCESS_KEY = os.getenv('MINIO_ACCESS_KEY', 'minioadmin')
MINIO_SECRET_KEY = os.getenv('MINIO_SECRET_KEY', 'minioadmin123')
MINIO_BUCKET_NAME = os.getenv('MINIO_BUCKET_NAME', 'lost-items-images')

# Initialize components
text_extractor = TextExtractor()
nlp_converter = NLPConverter()
rabbitmq_handler = None
minio_handler = None


def process_message(ch, method, properties, body):
    """
    Process incoming message from RabbitMQ.
    
    Expected message format:
    {
        "dataset_id": "uuid",
        "file_data": "base64-encoded-file-content",
        "file_name": "filename.csv",
        "file_format": ".csv"
    }
    """
    try:
        # Parse message
        message = json.loads(body.decode('utf-8'))
        logger.info(f"📨 Received message: {message.get('file_name', 'unknown')}")
        
        dataset_id = message.get('dataset_id')
        file_data = message.get('file_data')
        file_name = message.get('file_name')
        file_format = message.get('file_format', Path(file_name).suffix if file_name else '.txt')
        
        if not file_data:
            logger.error("No file_data in message")
            rabbitmq_handler.nack_message(method.delivery_tag, requeue=False)
            return
        
        # Decode file data
        try:
            file_content = base64.b64decode(file_data)
        except Exception as e:
            logger.error(f"Failed to decode file data: {e}")
            rabbitmq_handler.nack_message(method.delivery_tag, requeue=False)
            return
        
        # Save to temporary file
        with tempfile.NamedTemporaryFile(delete=False, suffix=file_format) as temp_file:
            temp_file.write(file_content)
            temp_path = temp_file.name
        
        try:
            # Extract text
            logger.info(f"📄 Extracting text from {file_name}...")
            extracted_data = text_extractor.extract(temp_path)
            
            # Convert to lost items
            logger.info(f"🔄 Converting to lost-items schema...")
            
            # Check if it's structured data with multiple items
            raw_data = extracted_data.get('raw_data')
            items = []
            
            if raw_data and isinstance(raw_data, list):
                # Multiple items (CSV rows)
                logger.info(f"Found {len(raw_data)} items in file")
                for i, row_data in enumerate(raw_data):
                    row_extracted = {
                        'text': extracted_data['text'].split('\n')[i] if i < len(extracted_data['text'].split('\n')) else '',
                        'raw_data': [row_data],
                        'metadata': extracted_data['metadata']
                    }
                    item = nlp_converter.convert(row_extracted, dataset_id)
                    item['id'] = str(uuid.uuid4())
                    item['created_at'] = datetime.utcnow().isoformat()
                    item['updated_at'] = datetime.utcnow().isoformat()
                    items.append(item)
            else:
                # Single item or unstructured text
                item = nlp_converter.convert(extracted_data, dataset_id)
                item['id'] = str(uuid.uuid4())
                item['created_at'] = datetime.utcnow().isoformat()
                item['updated_at'] = datetime.utcnow().isoformat()
                items.append(item)
            
            # Publish each item to RabbitMQ
            logger.info(f"📤 Publishing {len(items)} items to queue...")
            published_count = 0
            
            for item in items:
                if rabbitmq_handler.publish_item(item):
                    published_count += 1
                else:
                    logger.error(f"Failed to publish item {item['id']}")
            
            logger.info(f"✅ Successfully published {published_count}/{len(items)} items")
            
            # Acknowledge message
            rabbitmq_handler.ack_message(method.delivery_tag)
            
        finally:
            # Clean up temp file
            try:
                os.unlink(temp_path)
            except Exception as e:
                logger.warning(f"Failed to delete temp file: {e}")
    
    except json.JSONDecodeError as e:
        logger.error(f"Invalid JSON in message: {e}")
        rabbitmq_handler.nack_message(method.delivery_tag, requeue=False)
    
    except Exception as e:
        logger.error(f"Error processing message: {e}")
        import traceback
        traceback.print_exc()
        # Requeue the message for retry
        rabbitmq_handler.nack_message(method.delivery_tag, requeue=True)





if __name__ == '__main__':
    logger.info("🚀 Starting legacy-data-converter service...")
    logger.info(f"RabbitMQ URL: {RABBITMQ_URL}")
    logger.info(f"Exchange: {RABBITMQ_EXCHANGE}")
    logger.info(f"MinIO Endpoint: {MINIO_ENDPOINT}")
    
    try:
        # Initialize RabbitMQ handler
        rabbitmq_handler = RabbitMQHandler(RABBITMQ_URL, RABBITMQ_EXCHANGE)
        
        # Initialize MinIO handler (for future image handling)
        minio_handler = MinIOHandler(
            MINIO_ENDPOINT,
            MINIO_ACCESS_KEY,
            MINIO_SECRET_KEY,
            MINIO_BUCKET_NAME
        )
        
        # Start consuming messages
        rabbitmq_handler.start_consuming(process_message)
        
    except KeyboardInterrupt:
        logger.info("Shutting down...")
    except Exception as e:
        logger.error(f"Fatal error: {e}")
        import traceback
        traceback.print_exc()
    finally:
        if rabbitmq_handler:
            rabbitmq_handler.stop_consuming()
