# Test publishing dataset to RabbitMQ

# Install pika if needed
pip install pika

# Test with CSV file (multiple items)
python test_publish.py examples/lost_items.csv test-csv-001

# Test with JSON file
python test_publish.py examples/items.json test-json-001

# Test with XML file
python test_publish.py examples/items.xml test-xml-001

# Test with TXT file
python test_publish.py examples/report.txt test-txt-001

# Check logs
docker logs -f odnalezione-legacy-converter

# Check RabbitMQ queues
docker exec odnalezione-rabbitmq rabbitmqctl list_queues

# Check messages in queue
docker exec odnalezione-rabbitmq rabbitmqadmin list queues name messages
