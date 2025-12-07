"""
Test script to publish a dataset file to RabbitMQ for processing.
"""

import pika
import json
import base64
import sys
from pathlib import Path

def publish_dataset(file_path: str, dataset_id: str = "test-001", rabbitmq_url: str = "amqp://admin:admin123@localhost:5674/"):
    """
    Publish a dataset file to RabbitMQ for processing.
    
    Args:
        file_path: Path to the file to process
        dataset_id: Optional dataset ID
        rabbitmq_url: RabbitMQ connection URL
    """
    file_path = Path(file_path)
    
    if not file_path.exists():
        print(f"❌ File not found: {file_path}")
        return False
    
    print(f"📁 Reading file: {file_path.name}")
    
    # Read and encode file
    with open(file_path, 'rb') as f:
        file_data = base64.b64encode(f.read()).decode('utf-8')
    
    # Create message
    message = {
        'dataset_id': dataset_id,
        'file_data': file_data,
        'file_name': file_path.name,
        'file_format': file_path.suffix
    }
    
    print(f"📦 Message size: {len(file_data)} bytes (base64)")
    print(f"🎯 Dataset ID: {dataset_id}")
    
    try:
        # Connect to RabbitMQ
        print(f"🔌 Connecting to RabbitMQ: {rabbitmq_url}")
        connection = pika.BlockingConnection(
            pika.URLParameters(rabbitmq_url)
        )
        channel = connection.channel()
        
        # Publish message
        print(f"📤 Publishing to exchange 'lost-found.events' with routing key 'dataset.uploaded'")
        channel.basic_publish(
            exchange='lost-found.events',
            routing_key='dataset.uploaded',
            body=json.dumps(message),
            properties=pika.BasicProperties(
                delivery_mode=2,  # Persistent
                content_type='application/json'
            )
        )
        
        connection.close()
        
        print(f"✅ Message published successfully!")
        print(f"\nℹ️  Check logs: docker logs -f odnalezione-legacy-converter")
        return True
        
    except Exception as e:
        print(f"❌ Error: {e}")
        return False


if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: python test_publish.py <file_path> [dataset_id] [rabbitmq_url]")
        print("\nExamples:")
        print("  python test_publish.py examples/lost_items.csv")
        print("  python test_publish.py examples/items.json test-002")
        print("  python test_publish.py examples/report.txt test-003 amqp://admin:admin123@localhost:5674/")
        sys.exit(1)
    
    file_path = sys.argv[1]
    dataset_id = sys.argv[2] if len(sys.argv) > 2 else "test-001"
    rabbitmq_url = sys.argv[3] if len(sys.argv) > 3 else "amqp://admin:admin123@localhost:5674/"
    
    success = publish_dataset(file_path, dataset_id, rabbitmq_url)
    sys.exit(0 if success else 1)
