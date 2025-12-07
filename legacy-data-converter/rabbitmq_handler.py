"""
RabbitMQ handler for legacy data converter.
Consumes files from q.datasets.process queue and publishes converted items.
"""

import json
import logging
import time
from typing import Dict, Any, Optional
import pika
from pika.adapters.blocking_connection import BlockingChannel
from pika.spec import Basic, BasicProperties

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class RabbitMQHandler:
    """Handle RabbitMQ connections and message processing."""
    
    def __init__(self, rabbitmq_url: str, exchange: str = "lost-found.events"):
        self.rabbitmq_url = rabbitmq_url
        self.exchange = exchange
        self.connection = None
        self.channel = None
        
        # Queue names
        self.input_queue = "q.datasets.process"
        self.output_routing_key = "item.submitted"
        
        self._connect()
    
    def _connect(self):
        """Establish connection to RabbitMQ."""
        max_retries = 5
        retry_delay = 5
        
        for attempt in range(max_retries):
            try:
                logger.info(f"Connecting to RabbitMQ (attempt {attempt + 1}/{max_retries})...")
                
                parameters = pika.URLParameters(self.rabbitmq_url)
                self.connection = pika.BlockingConnection(parameters)
                self.channel = self.connection.channel()
                
                # Declare exchange
                self.channel.exchange_declare(
                    exchange=self.exchange,
                    exchange_type='topic',
                    durable=True
                )
                
                # Declare input queue
                self.channel.queue_declare(
                    queue=self.input_queue,
                    durable=True
                )
                
                # Bind queue to exchange (if needed)
                # Note: Queue should already be bound via rabbitmq-init.sh
                # Binding with routing key "dataset.uploaded"
                
                logger.info("✅ Connected to RabbitMQ successfully")
                return
                
            except Exception as e:
                logger.error(f"Failed to connect to RabbitMQ: {e}")
                if attempt < max_retries - 1:
                    logger.info(f"Retrying in {retry_delay} seconds...")
                    time.sleep(retry_delay)
                else:
                    raise Exception("Could not connect to RabbitMQ after multiple attempts")
    
    def publish_item(self, item: Dict[str, Any]) -> bool:
        """
        Publish a converted item to the processing pipeline.
        
        Args:
            item: Converted lost item
            
        Returns:
            True if published successfully
        """
        try:
            # Ensure connection is alive
            if not self.connection or self.connection.is_closed:
                logger.warning("Connection closed, reconnecting...")
                self._connect()
            
            # Prepare message
            message = {
                "item_id": item.get("id"),
                "text": item.get("title", ""),
                "description": item.get("description", ""),
                "category": item.get("category", ""),
                "location": item.get("location", ""),
                "date_lost": item.get("found_date"),
                "reporting_date": item.get("reporting_date"),
                "reporting_location": item.get("reporting_location", ""),
                "image_url": item.get("image_url", ""),
                "image_key": item.get("image_key", ""),
                "contact_email": item.get("contact_email", ""),
                "contact_phone": item.get("contact_phone", ""),
                "timestamp": item.get("created_at")
            }
            
            # Remove None values
            message = {k: v for k, v in message.items() if v is not None}
            
            # Publish to exchange
            self.channel.basic_publish(
                exchange=self.exchange,
                routing_key=self.output_routing_key,
                body=json.dumps(message),
                properties=pika.BasicProperties(
                    delivery_mode=2,  # Persistent
                    content_type='application/json'
                )
            )
            
            logger.info(f"✅ Published item {item.get('id')} to {self.output_routing_key}")
            return True
            
        except Exception as e:
            logger.error(f"Failed to publish item: {e}")
            return False
    
    def start_consuming(self, callback):
        """
        Start consuming messages from the input queue.
        
        Args:
            callback: Function to call for each message
        """
        try:
            logger.info(f"Starting to consume from {self.input_queue}...")
            
            # Set QoS
            self.channel.basic_qos(prefetch_count=1)
            
            # Start consuming
            self.channel.basic_consume(
                queue=self.input_queue,
                on_message_callback=callback,
                auto_ack=False
            )
            
            logger.info("✅ Waiting for messages. To exit press CTRL+C")
            self.channel.start_consuming()
            
        except KeyboardInterrupt:
            logger.info("Interrupted by user")
            self.stop_consuming()
        except Exception as e:
            logger.error(f"Error while consuming: {e}")
            raise
    
    def stop_consuming(self):
        """Stop consuming messages."""
        if self.channel:
            self.channel.stop_consuming()
        
        if self.connection and not self.connection.is_closed:
            self.connection.close()
        
        logger.info("Connection closed")
    
    def ack_message(self, delivery_tag: int):
        """Acknowledge a message."""
        if self.channel:
            self.channel.basic_ack(delivery_tag=delivery_tag)
    
    def nack_message(self, delivery_tag: int, requeue: bool = True):
        """Negative acknowledge a message."""
        if self.channel:
            self.channel.basic_nack(delivery_tag=delivery_tag, requeue=requeue)
