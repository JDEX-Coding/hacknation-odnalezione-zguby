"""
Message Format Converter
Ensures consistent format handling for incoming messages, supporting both
legacy image_url format and new image_key format for MinIO.
"""

import logging
from typing import Dict, Any, Optional, Tuple
from urllib.parse import urlparse
import re

logger = logging.getLogger(__name__)


class MessageConverter:
    """Converts and normalizes message formats for consistent processing."""

    def __init__(self):
        """Initialize the message converter."""
        self.stats = {
            'total_processed': 0,
            'url_format': 0,
            'key_format': 0,
            'no_image': 0,
            'errors': 0
        }

    def normalize_message(self, message: Dict[str, Any]) -> Dict[str, Any]:
        """
        Normalize incoming message to consistent format.

        Handles:
        - Legacy image_url format (HTTP/HTTPS URLs)
        - New image_key format (MinIO object keys)
        - Missing image fields
        - Field validation and cleanup

        Args:
            message: Raw message dictionary

        Returns:
            Normalized message with standardized fields
        """
        try:
            self.stats['total_processed'] += 1
            normalized = message.copy()

            # Handle image field conversion
            image_key = self._extract_image_key(message)
            normalized['image_key'] = image_key

            # Remove legacy image_url field to avoid confusion
            if 'image_url' in normalized:
                del normalized['image_url']

            # Normalize text fields
            normalized['text'] = self._normalize_text(message.get('text', ''))
            normalized['description'] = self._normalize_text(message.get('description', ''))
            normalized['category'] = self._normalize_text(message.get('category', ''))
            normalized['location'] = self._normalize_text(message.get('location', ''))

            # Ensure required fields exist
            normalized.setdefault('item_id', '')
            normalized.setdefault('date_lost', '')

            # Combine contact info if not present
            if not normalized.get('contact_info'):
                email = normalized.get('contact_email', '')
                phone = normalized.get('contact_phone', '')
                parts = []
                if email: parts.append(str(email))
                if phone: parts.append(str(phone))
                normalized['contact_info'] = ', '.join(parts)
            else:
                 normalized.setdefault('contact_info', '')

            logger.debug(f"Normalized message for item: {normalized.get('item_id', 'unknown')}")
            return normalized

        except Exception as e:
            self.stats['errors'] += 1
            logger.error(f"Error normalizing message: {e}")
            return message

    def _extract_image_key(self, message: Dict[str, Any]) -> str:
        """
        Extract and convert image reference to MinIO key format.

        Priority:
        1. image_key (native MinIO format)
        2. image_url (convert to MinIO key if possible, or log warning)
        3. Empty string (no image)

        Args:
            message: Message dictionary

        Returns:
            MinIO object key or empty string
        """
        # Check for native image_key format
        if 'image_key' in message and message['image_key']:
            image_key = str(message['image_key']).strip()
            if image_key:
                self.stats['key_format'] += 1
                logger.debug(f"Using native image_key: {image_key}")
                return self._normalize_object_key(image_key)

        # Check for legacy image_url format
        if 'image_url' in message and message['image_url']:
            image_url = str(message['image_url']).strip()
            if image_url:
                # Try to convert URL to object key
                object_key = self._url_to_object_key(image_url)
                if object_key:
                    self.stats['url_format'] += 1
                    logger.warning(
                        f"Converted legacy image_url to image_key: {image_url} -> {object_key}. "
                        "Please update message producers to use image_key format."
                    )
                    return object_key
                else:
                    logger.error(
                        f"Cannot convert image_url to object key: {image_url}. "
                        "Image will be skipped. Please upload to MinIO and use image_key."
                    )
                    self.stats['errors'] += 1
                    return ''

        # No image reference found
        self.stats['no_image'] += 1
        return ''

    def _url_to_object_key(self, url: str) -> Optional[str]:
        """
        Attempt to convert a URL to a MinIO object key.

        This is a best-effort conversion for backward compatibility.
        Strategies:
        1. Check if it looks like a MinIO URL and extract the object key
        2. Extract filename from URL path
        3. Return None if conversion not possible

        Args:
            url: Image URL

        Returns:
            MinIO object key or None
        """
        try:
            parsed = urlparse(url)

            # Strategy 1: Check if it's already a MinIO URL pattern
            # e.g., http://minio:9000/lost-items-images/items/item123.jpg
            if 'minio' in parsed.netloc.lower() or parsed.port == 9000:
                # Extract path after bucket name
                path_parts = parsed.path.strip('/').split('/', 1)
                if len(path_parts) > 1:
                    # Remove bucket name, keep object key
                    return path_parts[1]

            # Strategy 2: Extract filename and create standard object key
            # e.g., https://example.com/images/backpack.jpg -> items/backpack.jpg
            if parsed.path:
                filename = parsed.path.split('/')[-1]
                if filename and '.' in filename:
                    # Create a standard items/ prefix
                    return f"items/{filename}"

            return None

        except Exception as e:
            logger.error(f"Error converting URL to object key: {e}")
            return None

    def _normalize_object_key(self, key: str) -> str:
        """
        Normalize MinIO object key format.

        - Remove leading/trailing slashes
        - Ensure valid characters
        - Handle path separators consistently

        Args:
            key: Raw object key

        Returns:
            Normalized object key
        """
        # Remove leading/trailing whitespace and slashes
        key = key.strip().strip('/')

        # Replace backslashes with forward slashes
        key = key.replace('\\', '/')

        # Remove duplicate slashes
        key = re.sub(r'/+', '/', key)

        return key

    def _normalize_text(self, text: Any) -> str:
        """
        Normalize text field.

        Args:
            text: Raw text value

        Returns:
            Cleaned text string
        """
        if text is None:
            return ''

        # Convert to string and strip whitespace
        text = str(text).strip()

        # Remove excessive whitespace
        text = re.sub(r'\s+', ' ', text)

        return text

    def validate_message(self, message: Dict[str, Any]) -> Tuple[bool, Optional[str]]:
        """
        Validate message has required fields and proper format.

        Args:
            message: Message dictionary

        Returns:
            Tuple of (is_valid, error_message)
        """
        # Check required fields
        if not message.get('item_id'):
            return False, "Missing required field: item_id"

        # Check that at least one text field exists
        if not any([
            message.get('text'),
            message.get('description'),
            message.get('category')
        ]):
            return False, "Message must contain at least one text field (text, description, or category)"

        # Validate image_key format if present
        image_key = message.get('image_key', '')
        if image_key:
            if not self._is_valid_object_key(image_key):
                return False, f"Invalid image_key format: {image_key}"

        return True, None

    def _is_valid_object_key(self, key: str) -> bool:
        """
        Check if object key is valid.

        Args:
            key: Object key to validate

        Returns:
            True if valid, False otherwise
        """
        if not key or not isinstance(key, str):
            return False

        # Check for invalid characters
        invalid_chars = ['\\', '\0', '\n', '\r', '\t']
        if any(char in key for char in invalid_chars):
            return False

        # Check length (S3 allows up to 1024 bytes)
        if len(key.encode('utf-8')) > 1024:
            return False

        return True

    def get_statistics(self) -> Dict[str, int]:
        """
        Get processing statistics.

        Returns:
            Dictionary with statistics
        """
        return self.stats.copy()

    def reset_statistics(self):
        """Reset all statistics counters."""
        self.stats = {
            'total_processed': 0,
            'url_format': 0,
            'key_format': 0,
            'no_image': 0,
            'errors': 0
        }

    def create_output_message(
        self,
        input_message: Dict[str, Any],
        embedding: list,
        has_image_embedding: bool,
        timestamp: str
    ) -> Dict[str, Any]:
        """
        Create standardized output message for vectorized data.

        Args:
            input_message: Original normalized input message
            embedding: Generated embedding vector
            has_image_embedding: Whether image was processed
            timestamp: Processing timestamp

        Returns:
            Standardized output message
        """
        return {
            'item_id': input_message.get('item_id', ''),
            'embedding': embedding,
            'title': input_message.get('text', ''),
            'description': input_message.get('description', ''),
            'category': input_message.get('category', ''),
            'location': input_message.get('location', ''),
            'date_lost': input_message.get('date_lost', ''),
            'image_key': input_message.get('image_key', ''),
            'contact_info': input_message.get('contact_info', ''),
            'timestamp': timestamp,
            'has_image_embedding': has_image_embedding
        }
