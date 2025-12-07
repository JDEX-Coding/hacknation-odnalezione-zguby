"""
Quick test script for CLIP service local setup
Tests basic functionality without needing full infrastructure
"""

import sys
from pathlib import Path

def test_imports():
    """Test that all required packages are installed."""
    print("🧪 Testing imports...")
    try:
        import torch
        import torchvision
        import transformers
        import PIL
        import pika
        import minio
        import numpy
        print("✅ All required packages imported successfully")

        print(f"   - PyTorch version: {torch.__version__}")
        print(f"   - Transformers version: {transformers.__version__}")
        print(f"   - CUDA available: {torch.cuda.is_available()}")
        return True
    except ImportError as e:
        print(f"❌ Import failed: {e}")
        return False

def test_clip_model():
    """Test CLIP model loading."""
    print("\n🧪 Testing CLIP model loading...")
    try:
        from clip_handler import CLIPHandler

        handler = CLIPHandler()
        handler.load_model()

        # Test text encoding
        text = "A black backpack"
        embedding = handler.encode_text(text)
        print(f"✅ Text encoding successful")
        print(f"   - Embedding shape: {embedding.shape}")
        print(f"   - Embedding dimensions: {len(embedding)}")

        return True
    except Exception as e:
        print(f"❌ CLIP model test failed: {e}")
        return False

def test_environment():
    """Test environment configuration."""
    print("\n🧪 Testing environment configuration...")
    try:
        from dotenv import load_dotenv
        import os

        load_dotenv()

        rabbitmq_url = os.getenv('RABBITMQ_URL', 'Not set')
        minio_endpoint = os.getenv('MINIO_ENDPOINT', 'Not set')

        print(f"✅ Environment variables loaded")
        print(f"   - RabbitMQ URL: {rabbitmq_url}")
        print(f"   - MinIO Endpoint: {minio_endpoint}")

        return True
    except Exception as e:
        print(f"❌ Environment test failed: {e}")
        return False

def test_cache_directory():
    """Test cache directory setup."""
    print("\n🧪 Testing cache directory...")
    try:
        cache_dir = Path(".cache/huggingface")
        if cache_dir.exists():
            print(f"✅ Cache directory exists: {cache_dir.absolute()}")
        else:
            cache_dir.mkdir(parents=True, exist_ok=True)
            print(f"✅ Cache directory created: {cache_dir.absolute()}")
        return True
    except Exception as e:
        print(f"❌ Cache directory test failed: {e}")
        return False

def main():
    """Run all tests."""
    print("=" * 60)
    print("🚀 CLIP Service Local Setup Test")
    print("=" * 60)

    tests = [
        ("Package Imports", test_imports),
        ("Environment Config", test_environment),
        ("Cache Directory", test_cache_directory),
        ("CLIP Model", test_clip_model),
    ]

    results = []
    for name, test_func in tests:
        try:
            result = test_func()
            results.append((name, result))
        except Exception as e:
            print(f"\n❌ Test '{name}' crashed: {e}")
            results.append((name, False))

    print("\n" + "=" * 60)
    print("📊 Test Summary")
    print("=" * 60)

    passed = sum(1 for _, result in results if result)
    total = len(results)

    for name, result in results:
        status = "✅ PASS" if result else "❌ FAIL"
        print(f"{status} - {name}")

    print(f"\nTotal: {passed}/{total} tests passed")

    if passed == total:
        print("\n🎉 All tests passed! Your CLIP service is ready for local development.")
        return 0
    else:
        print("\n⚠️  Some tests failed. Please check the errors above.")
        return 1

if __name__ == "__main__":
    sys.exit(main())
