"""
MinIO handler for legacy data converter.
Handles file storage and retrieval from MinIO.
"""

import logging
from minio import Minio
from minio.error import S3Error
import time

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class MinIOHandler:
    """Handle MinIO storage operations."""
    
    def __init__(self, endpoint: str, access_key: str, secret_key: str, bucket_name: str, secure: bool = False):
        self.endpoint = endpoint
        self.access_key = access_key
        self.secret_key = secret_key
        self.bucket_name = bucket_name
        self.secure = secure
        self.client = None
        
        self._connect()
    
    def _connect(self):
        """Establish connection to MinIO."""
        max_retries = 5
        retry_delay = 5
        
        for attempt in range(max_retries):
            try:
                logger.info(f"Connecting to MinIO at {self.endpoint} (attempt {attempt + 1}/{max_retries})...")
                
                self.client = Minio(
                    self.endpoint,
                    access_key=self.access_key,
                    secret_key=self.secret_key,
                    secure=self.secure
                )
                
                # Check if bucket exists
                if not self.client.bucket_exists(self.bucket_name):
                    logger.warning(f"Bucket {self.bucket_name} does not exist, creating...")
                    self.client.make_bucket(self.bucket_name)
                
                logger.info("✅ Connected to MinIO successfully")
                return
                
            except Exception as e:
                logger.error(f"Failed to connect to MinIO: {e}")
                if attempt < max_retries - 1:
                    logger.info(f"Retrying in {retry_delay} seconds...")
                    time.sleep(retry_delay)
                else:
                    logger.warning("Could not connect to MinIO, continuing without storage")
                    self.client = None
    
    def upload_file(self, file_path: str, object_name: str) -> str:
        """
        Upload a file to MinIO.
        
        Args:
            file_path: Path to local file
            object_name: Name for object in MinIO
            
        Returns:
            URL to the uploaded file
        """
        try:
            if not self.client:
                raise Exception("MinIO client not initialized")
            
            self.client.fput_object(
                self.bucket_name,
                object_name,
                file_path
            )
            
            url = f"http://{self.endpoint}/{self.bucket_name}/{object_name}"
            logger.info(f"✅ Uploaded file to MinIO: {url}")
            return url
            
        except S3Error as e:
            logger.error(f"MinIO S3 error: {e}")
            raise
        except Exception as e:
            logger.error(f"Failed to upload file to MinIO: {e}")
            raise
    
    def get_file_url(self, object_name: str) -> str:
        """
        Get URL for an object in MinIO.
        
        Args:
            object_name: Name of object
            
        Returns:
            URL to the object
        """
        return f"http://{self.endpoint}/{self.bucket_name}/{object_name}"
