"""
MinIO Handler
Handles downloading images from MinIO object storage for processing.
"""

import os
import logging
import io
from pathlib import Path
from typing import Optional

from minio import Minio
from minio.error import S3Error
from PIL import Image

logger = logging.getLogger(__name__)


class MinIOHandler:
    """Handles downloading and managing images from MinIO object storage."""
    
    def __init__(
        self,
        endpoint: Optional[str] = None,
        access_key: Optional[str] = None,
        secret_key: Optional[str] = None,
        bucket_name: str = "lost-items-images",
        tmp_dir: str = "/tmp/clip-images",
        secure: bool = False
    ):
        """
        Initialize MinIO handler.
        
        Args:
            endpoint: MinIO endpoint (e.g., 'minio:9000')
            access_key: MinIO access key
            secret_key: MinIO secret key
            bucket_name: Name of the bucket containing images
            tmp_dir: Directory to store temporary images
            secure: Whether to use HTTPS (default: False for local development)
        """
        self.endpoint = endpoint or os.getenv('MINIO_ENDPOINT', 'minio:9000')
        self.access_key = access_key or os.getenv('MINIO_ACCESS_KEY', 'minioadmin')
        self.secret_key = secret_key or os.getenv('MINIO_SECRET_KEY', 'minioadmin123')
        self.bucket_name = bucket_name or os.getenv('MINIO_BUCKET_NAME', 'lost-items-images')
        self.secure = secure
        self.tmp_dir = Path(tmp_dir)
        
        # Create tmp directory if it doesn't exist
        self.tmp_dir.mkdir(parents=True, exist_ok=True)
        
        # Initialize MinIO client
        try:
            self.client = Minio(
                self.endpoint,
                access_key=self.access_key,
                secret_key=self.secret_key,
                secure=self.secure
            )
            logger.info(f"MinIO client initialized: {self.endpoint}")
            
            # Verify bucket exists
            if not self.client.bucket_exists(self.bucket_name):
                logger.warning(f"Bucket '{self.bucket_name}' does not exist")
            else:
                logger.info(f"Connected to MinIO bucket: {self.bucket_name}")
                
        except Exception as e:
            logger.error(f"Failed to initialize MinIO client: {e}")
            raise
    
    def download_image(
        self,
        object_name: str,
        item_id: str,
        max_size_mb: int = 10
    ) -> Optional[Path]:
        """
        Download an image from MinIO.
        
        Args:
            object_name: Name/key of the object in MinIO (e.g., 'item123.jpg' or 'images/item123.jpg')
            item_id: Unique identifier for the item (used in local filename)
            max_size_mb: Maximum allowed file size in MB
            
        Returns:
            Path to the downloaded image, or None if download failed
        """
        try:
            logger.info(f"Downloading image from MinIO: {object_name}")
            
            # Get object stats to check size
            try:
                stat = self.client.stat_object(self.bucket_name, object_name)
                size_mb = stat.size / (1024 * 1024)
                
                if size_mb > max_size_mb:
                    logger.error(f"Image too large: {size_mb:.2f}MB (max: {max_size_mb}MB)")
                    return None
                    
                logger.debug(f"Image size: {size_mb:.2f}MB")
                
            except S3Error as e:
                logger.error(f"Object not found or error getting stats: {e}")
                return None
            
            # Determine file extension from object name
            file_extension = Path(object_name).suffix or '.jpg'
            filename = f"{item_id}_minio{file_extension}"
            file_path = self.tmp_dir / filename
            
            # Download object to file
            self.client.fget_object(
                self.bucket_name,
                object_name,
                str(file_path)
            )
            
            # Validate image can be opened
            try:
                img = Image.open(file_path)
                img.verify()
                logger.info(f"✅ Image downloaded successfully from MinIO: {file_path.name}")
                return file_path
            except Exception as e:
                logger.error(f"Downloaded file is not a valid image: {e}")
                self.cleanup_image(file_path)
                return None
            
        except S3Error as e:
            logger.error(f"MinIO error downloading image: {e}")
            return None
        except Exception as e:
            logger.error(f"Unexpected error downloading image from MinIO: {e}")
            return None
    
    def download_image_to_memory(self, object_name: str) -> Optional[Image.Image]:
        """
        Download an image from MinIO directly to memory without saving to disk.
        
        Args:
            object_name: Name/key of the object in MinIO
            
        Returns:
            PIL Image object, or None if download failed
        """
        try:
            logger.info(f"Downloading image to memory from MinIO: {object_name}")
            
            # Get object
            response = self.client.get_object(self.bucket_name, object_name)
            
            # Read data into memory
            image_data = response.read()
            response.close()
            response.release_conn()
            
            # Open as PIL Image
            img = Image.open(io.BytesIO(image_data))
            img.verify()
            
            # Re-open for actual use (verify closes the image)
            img = Image.open(io.BytesIO(image_data))
            logger.info(f"✅ Image loaded to memory successfully: {object_name}")
            return img
            
        except S3Error as e:
            logger.error(f"MinIO error downloading image to memory: {e}")
            return None
        except Exception as e:
            logger.error(f"Unexpected error downloading image to memory: {e}")
            return None
    
    def cleanup_image(self, file_path: Path) -> bool:
        """
        Delete a temporary image file.
        
        Args:
            file_path: Path to the file to delete
            
        Returns:
            True if deleted successfully, False otherwise
        """
        try:
            if file_path.exists():
                file_path.unlink()
                logger.debug(f"Cleaned up image: {file_path.name}")
                return True
            return False
        except Exception as e:
            logger.error(f"Error cleaning up image {file_path}: {e}")
            return False
    
    def cleanup_old_images(self, max_age_hours: int = 24):
        """
        Clean up old temporary images.
        
        Args:
            max_age_hours: Maximum age of files to keep in hours
        """
        import time
        
        try:
            current_time = time.time()
            max_age_seconds = max_age_hours * 3600
            
            deleted_count = 0
            for file_path in self.tmp_dir.iterdir():
                if file_path.is_file():
                    file_age = current_time - file_path.stat().st_mtime
                    if file_age > max_age_seconds:
                        self.cleanup_image(file_path)
                        deleted_count += 1
            
            if deleted_count > 0:
                logger.info(f"Cleaned up {deleted_count} old image(s)")
                
        except Exception as e:
            logger.error(f"Error during cleanup: {e}")
    
    def list_objects(self, prefix: str = "") -> list:
        """
        List objects in the bucket.
        
        Args:
            prefix: Filter objects by prefix
            
        Returns:
            List of object names
        """
        try:
            objects = self.client.list_objects(self.bucket_name, prefix=prefix)
            object_names = [obj.object_name for obj in objects]
            logger.info(f"Found {len(object_names)} objects with prefix '{prefix}'")
            return object_names
        except Exception as e:
            logger.error(f"Error listing objects: {e}")
            return []
    
    def object_exists(self, object_name: str) -> bool:
        """
        Check if an object exists in the bucket.
        
        Args:
            object_name: Name/key of the object
            
        Returns:
            True if object exists, False otherwise
        """
        try:
            self.client.stat_object(self.bucket_name, object_name)
            return True
        except S3Error:
            return False
        except Exception as e:
            logger.error(f"Error checking object existence: {e}")
            return False
    
    def get_image_info(self, object_name: str) -> dict:
        """
        Get information about an image in MinIO.
        
        Args:
            object_name: Name/key of the object
            
        Returns:
            Dictionary with image information
        """
        try:
            stat = self.client.stat_object(self.bucket_name, object_name)
            return {
                'object_name': object_name,
                'size_bytes': stat.size,
                'size_mb': stat.size / (1024 * 1024),
                'etag': stat.etag,
                'last_modified': stat.last_modified,
                'content_type': stat.content_type
            }
        except Exception as e:
            logger.error(f"Error getting object info: {e}")
            return {}
