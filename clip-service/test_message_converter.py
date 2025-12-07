"""
Unit tests for MessageConverter
Tests format conversion, validation, and normalization.
"""

import unittest
from message_converter import MessageConverter


class TestMessageConverter(unittest.TestCase):
    """Test cases for MessageConverter class."""
    
    def setUp(self):
        """Set up test fixtures."""
        self.converter = MessageConverter()
    
    def test_normalize_message_with_image_key(self):
        """Test normalization with native image_key format."""
        message = {
            'item_id': 'test123',
            'text': 'Lost backpack',
            'description': 'Blue backpack',
            'category': 'Bags',
            'image_key': 'items/test123.jpg',
            'location': 'Station',
            'date_lost': '2024-12-06',
            'contact_info': 'test@example.com'
        }
        
        normalized = self.converter.normalize_message(message)
        
        self.assertEqual(normalized['item_id'], 'test123')
        self.assertEqual(normalized['image_key'], 'items/test123.jpg')
        self.assertNotIn('image_url', normalized)
    
    def test_normalize_message_with_legacy_url(self):
        """Test backward compatibility with legacy image_url format."""
        message = {
            'item_id': 'test456',
            'text': 'Lost wallet',
            'image_url': 'https://example.com/images/wallet.jpg'
        }
        
        normalized = self.converter.normalize_message(message)
        
        self.assertEqual(normalized['item_id'], 'test456')
        self.assertEqual(normalized['image_key'], 'items/wallet.jpg')
        self.assertNotIn('image_url', normalized)
    
    def test_normalize_message_with_minio_url(self):
        """Test conversion of MinIO-style URLs."""
        message = {
            'item_id': 'test789',
            'text': 'Lost keys',
            'image_url': 'http://minio:9000/lost-items-images/items/keys.jpg'
        }
        
        normalized = self.converter.normalize_message(message)
        
        self.assertEqual(normalized['image_key'], 'items/keys.jpg')
    
    def test_normalize_message_no_image(self):
        """Test normalization when no image is provided."""
        message = {
            'item_id': 'test000',
            'text': 'Lost phone',
            'description': 'iPhone 12'
        }
        
        normalized = self.converter.normalize_message(message)
        
        self.assertEqual(normalized['image_key'], '')
        self.assertEqual(self.converter.stats['no_image'], 1)
    
    def test_normalize_text_fields(self):
        """Test text normalization removes extra whitespace."""
        message = {
            'item_id': 'test111',
            'text': '  Lost   backpack  ',
            'description': 'Contains\n\nlaptop',
            'category': '  Bags  '
        }
        
        normalized = self.converter.normalize_message(message)
        
        self.assertEqual(normalized['text'], 'Lost backpack')
        self.assertEqual(normalized['description'], 'Contains laptop')
        self.assertEqual(normalized['category'], 'Bags')
    
    def test_normalize_object_key(self):
        """Test object key normalization."""
        # Test with leading/trailing slashes
        self.assertEqual(
            self.converter._normalize_object_key('/items/test.jpg/'),
            'items/test.jpg'
        )
        
        # Test with backslashes
        self.assertEqual(
            self.converter._normalize_object_key('items\\test.jpg'),
            'items/test.jpg'
        )
        
        # Test with duplicate slashes
        self.assertEqual(
            self.converter._normalize_object_key('items//test///file.jpg'),
            'items/test/file.jpg'
        )
    
    def test_validate_message_success(self):
        """Test validation passes for valid message."""
        message = {
            'item_id': 'test222',
            'text': 'Lost item',
            'image_key': 'items/test.jpg'
        }
        
        is_valid, error = self.converter.validate_message(message)
        
        self.assertTrue(is_valid)
        self.assertIsNone(error)
    
    def test_validate_message_missing_item_id(self):
        """Test validation fails without item_id."""
        message = {
            'text': 'Lost item'
        }
        
        is_valid, error = self.converter.validate_message(message)
        
        self.assertFalse(is_valid)
        self.assertIn('item_id', error)
    
    def test_validate_message_missing_text_fields(self):
        """Test validation fails without any text fields."""
        message = {
            'item_id': 'test333'
        }
        
        is_valid, error = self.converter.validate_message(message)
        
        self.assertFalse(is_valid)
        self.assertIn('text field', error)
    
    def test_validate_message_invalid_image_key(self):
        """Test validation fails with invalid image_key."""
        message = {
            'item_id': 'test444',
            'text': 'Lost item',
            'image_key': 'invalid\x00key'
        }
        
        is_valid, error = self.converter.validate_message(message)
        
        self.assertFalse(is_valid)
        self.assertIn('Invalid image_key', error)
    
    def test_url_to_object_key_standard_url(self):
        """Test conversion of standard image URL to object key."""
        url = 'https://example.com/images/photo.jpg'
        key = self.converter._url_to_object_key(url)
        
        self.assertEqual(key, 'items/photo.jpg')
    
    def test_url_to_object_key_minio_url(self):
        """Test extraction of key from MinIO URL."""
        url = 'http://minio:9000/lost-items-images/uploads/photo.jpg'
        key = self.converter._url_to_object_key(url)
        
        self.assertEqual(key, 'uploads/photo.jpg')
    
    def test_url_to_object_key_invalid(self):
        """Test that invalid URLs return None."""
        url = 'not-a-valid-url'
        key = self.converter._url_to_object_key(url)
        
        self.assertIsNone(key)
    
    def test_is_valid_object_key(self):
        """Test object key validation."""
        # Valid keys
        self.assertTrue(self.converter._is_valid_object_key('items/test.jpg'))
        self.assertTrue(self.converter._is_valid_object_key('folder/subfolder/file.png'))
        
        # Invalid keys
        self.assertFalse(self.converter._is_valid_object_key(''))
        self.assertFalse(self.converter._is_valid_object_key('key\x00with\x00nulls'))
        self.assertFalse(self.converter._is_valid_object_key('key\nwith\nnewlines'))
        self.assertFalse(self.converter._is_valid_object_key('a' * 2000))  # Too long
    
    def test_create_output_message(self):
        """Test output message creation."""
        input_msg = {
            'item_id': 'test555',
            'text': 'Lost item',
            'description': 'Description',
            'category': 'Category',
            'location': 'Location',
            'date_lost': '2024-12-06',
            'image_key': 'items/test.jpg',
            'contact_info': 'contact@example.com'
        }
        
        embedding = [0.1, 0.2, 0.3]
        timestamp = '2024-12-06T12:00:00'
        
        output = self.converter.create_output_message(
            input_msg,
            embedding,
            has_image_embedding=True,
            timestamp=timestamp
        )
        
        self.assertEqual(output['item_id'], 'test555')
        self.assertEqual(output['embedding'], embedding)
        self.assertEqual(output['title'], 'Lost item')
        self.assertTrue(output['has_image_embedding'])
        self.assertEqual(output['timestamp'], timestamp)
    
    def test_statistics_tracking(self):
        """Test that statistics are tracked correctly."""
        self.converter.reset_statistics()
        
        # Process message with image_key
        msg1 = {'item_id': 'test1', 'text': 'Test', 'image_key': 'items/test.jpg'}
        self.converter.normalize_message(msg1)
        
        # Process message with image_url
        msg2 = {'item_id': 'test2', 'text': 'Test', 'image_url': 'http://example.com/img.jpg'}
        self.converter.normalize_message(msg2)
        
        # Process message without image
        msg3 = {'item_id': 'test3', 'text': 'Test'}
        self.converter.normalize_message(msg3)
        
        stats = self.converter.get_statistics()
        
        self.assertEqual(stats['total_processed'], 3)
        self.assertEqual(stats['key_format'], 1)
        self.assertEqual(stats['url_format'], 1)
        self.assertEqual(stats['no_image'], 1)
    
    def test_priority_image_key_over_url(self):
        """Test that image_key takes priority over image_url."""
        message = {
            'item_id': 'test666',
            'text': 'Lost item',
            'image_key': 'items/priority.jpg',
            'image_url': 'https://example.com/fallback.jpg'
        }
        
        normalized = self.converter.normalize_message(message)
        
        # Should use image_key, not convert from image_url
        self.assertEqual(normalized['image_key'], 'items/priority.jpg')


if __name__ == '__main__':
    unittest.main()
