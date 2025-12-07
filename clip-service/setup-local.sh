#!/bin/bash
# Bash script to set up CLIP service for local development (Linux/Mac)
# Run this from the clip-service directory

echo "🚀 Setting up CLIP Service for Local Development"
echo ""

# Check Python version
echo "Checking Python version..."
python3 --version
echo ""

# Create virtual environment if it doesn't exist
if [ ! -d "venv" ]; then
    echo "Creating virtual environment..."
    python3 -m venv venv
    echo "✅ Virtual environment created"
else
    echo "✅ Virtual environment already exists"
fi
echo ""

# Activate virtual environment
echo "Activating virtual environment..."
source venv/bin/activate

# Upgrade pip
echo "Upgrading pip..."
python -m pip install --upgrade pip
echo ""

# Install dependencies
echo "Installing dependencies (CPU-only PyTorch for faster install)..."
pip install -r requirements-dev.txt
echo "✅ Dependencies installed"
echo ""

# Create .env file if it doesn't exist
if [ ! -f ".env" ]; then
    echo "Creating .env file from .env.local..."
    cp .env.local .env
    echo "✅ .env file created"
else
    echo "⚠️  .env file already exists, skipping..."
fi
echo ""

# Create cache directory
echo "Creating cache directory..."
mkdir -p .cache/huggingface
echo "✅ Cache directory created"
echo ""

# Pre-download CLIP model (optional but recommended)
read -p "Do you want to pre-download the CLIP model now? (y/N) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Downloading CLIP model (this may take a few minutes)..."
    python -c "from transformers import CLIPProcessor, CLIPModel; CLIPProcessor.from_pretrained('openai/clip-vit-base-patch32'); CLIPModel.from_pretrained('openai/clip-vit-base-patch32'); print('✅ Model downloaded successfully')"
    echo ""
fi

echo "🎉 Setup complete!"
echo ""
echo "Next steps:"
echo "1. Make sure RabbitMQ is running (docker-compose up rabbitmq -d)"
echo "2. Make sure MinIO is running (docker-compose up minio -d)"
echo "3. Run the service: python main.py"
echo ""
echo "Or run everything with: docker-compose up rabbitmq minio -d"
