#!/bin/bash
# Fix MinIO Bucket Policy for Public Access
# Run this on your production server to fix the bucket policy

echo "🔧 Fixing MinIO bucket policy for public image access..."

# Get environment variables
source .env.production 2>/dev/null || source .env 2>/dev/null

BUCKET_NAME="${MINIO_BUCKET_NAME:-lost-items-images}"
ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin123}"

echo "📦 Bucket: $BUCKET_NAME"

# Run mc commands in the minio container or through docker-compose
docker-compose exec minio sh -c "
    mc alias set local http://localhost:9000 $ACCESS_KEY $SECRET_KEY
    echo '✅ Connected to MinIO'

    mc anonymous set public local/$BUCKET_NAME
    echo '✅ Bucket policy set to public'

    mc anonymous get local/$BUCKET_NAME
    echo '📋 Current policy displayed above'
"

echo ""
echo "🎉 Done! Your images should now be accessible at:"
echo "   https://minio.jdex.pl/$BUCKET_NAME/uploads/..."
echo ""
echo "Test with:"
echo "   curl -I https://minio.jdex.pl/$BUCKET_NAME/uploads/2025-12-07/8b31fffe-c949-4890-9337-ff2ab3cee9a0.jpg"
