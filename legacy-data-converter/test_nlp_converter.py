"""
Tests for nlp_converter module.
"""

import pytest
from datetime import datetime
from nlp_converter import NLPConverter, convert_text_to_item


@pytest.fixture
def converter():
    return NLPConverter()


@pytest.fixture
def sample_extracted_data():
    return {
        "text": """
        Znaleziono portfel przy ulicy Marszałkowskiej 15.
        Data znalezienia: 15.01.2024
        Kontakt: biuro@urzad.pl, tel: 123-456-789
        """,
        "metadata": {
            "file_name": "test.txt",
            "file_format": ".txt"
        },
        "raw_data": None
    }


@pytest.fixture
def sample_structured_data():
    return {
        "text": "Structured data",
        "metadata": {
            "file_name": "test.csv",
            "file_format": ".csv"
        },
        "raw_data": [{
            "title": "Portfel skórzany",
            "description": "Czarny portfel z dokumentami",
            "category": "portfele",
            "location": "ul. Marszałkowska 1",
            "found_date": "2024-01-15",
            "email": "biuro@urzad.pl"
        }]
    }


def test_converter_initialization(converter):
    """Test NLPConverter initialization."""
    assert converter is not None
    assert len(converter.CATEGORIES) > 0
    assert "dokumenty" in converter.CATEGORIES


def test_convert_unstructured_text(converter, sample_extracted_data):
    """Test conversion of unstructured text."""
    result = converter.convert(sample_extracted_data)
    
    assert result is not None
    assert 'title' in result
    assert 'description' in result
    assert 'category' in result
    assert result['status'] == 'pending'


def test_convert_structured_data(converter, sample_structured_data):
    """Test conversion of structured data."""
    result = converter.convert(sample_structured_data)
    
    assert result is not None
    assert result['title'] == "Portfel skórzany"
    assert result['category'] == "portfele"
    assert result['location'] == "ul. Marszałkowska 1"
    assert result['contact_email'] == "biuro@urzad.pl"


def test_extract_email(converter):
    """Test email extraction."""
    text = "Contact us at test@example.com for more information."
    email = converter._extract_email(text)
    assert email == "test@example.com"


def test_extract_phone(converter):
    """Test phone number extraction."""
    text = "Call us at 123-456-789 or +48 123 456 789"
    phone = converter._extract_phone(text)
    assert phone != ""


def test_extract_category(converter):
    """Test category extraction."""
    text = "Znaleziono telefon Samsung Galaxy"
    category = converter._extract_category(text)
    assert category in ["telefony", "elektronika"]


def test_parse_date(converter):
    """Test date parsing."""
    # Test different date formats
    assert converter._parse_date("15.01.2024") is not None
    assert converter._parse_date("2024-01-15") is not None
    assert converter._parse_date("15 stycznia 2024") is not None


def test_create_empty_item(converter):
    """Test empty item creation."""
    item = converter._create_empty_item()
    assert item['status'] == 'pending'
    assert item['category'] == 'inne'
    assert item['title'] == ''


def test_convert_with_dataset_id(converter, sample_extracted_data):
    """Test conversion with dataset ID."""
    dataset_id = "test-dataset-123"
    result = converter.convert(sample_extracted_data, dataset_id)
    
    assert result['dataset_id'] == dataset_id


def test_convert_batch(converter):
    """Test batch conversion."""
    items = [
        {
            "text": "Znaleziono portfel",
            "metadata": {"file_name": "file1.txt", "file_format": ".txt"},
            "raw_data": None
        },
        {
            "text": "Znaleziono telefon",
            "metadata": {"file_name": "file2.txt", "file_format": ".txt"},
            "raw_data": None
        }
    ]
    
    results = converter.convert_batch(items)
    assert len(results) == 2
    assert all('title' in item for item in results)


def test_convenience_function(sample_extracted_data):
    """Test convenience function."""
    result = convert_text_to_item(sample_extracted_data)
    assert result is not None
    assert 'title' in result
