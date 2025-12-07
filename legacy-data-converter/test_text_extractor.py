"""
Tests for text_extractor module.
"""

import pytest
import json
from pathlib import Path
from text_extractor import TextExtractor, extract_from_file


@pytest.fixture
def extractor():
    return TextExtractor()


@pytest.fixture
def test_files_dir(tmp_path):
    """Create temporary test files."""
    test_dir = tmp_path / "test_files"
    test_dir.mkdir()
    
    # Create test TXT file
    txt_file = test_dir / "test.txt"
    txt_file.write_text("This is a test text file.\nIt has multiple lines.", encoding='utf-8')
    
    # Create test JSON file
    json_file = test_dir / "test.json"
    json_data = {
        "title": "Lost wallet",
        "description": "Black leather wallet",
        "location": "Main Street"
    }
    json_file.write_text(json.dumps(json_data, indent=2), encoding='utf-8')
    
    # Create test HTML file
    html_file = test_dir / "test.html"
    html_content = """
    <html>
    <head><title>Test</title></head>
    <body>
        <h1>Lost Items</h1>
        <p>This is a test HTML document.</p>
    </body>
    </html>
    """
    html_file.write_text(html_content, encoding='utf-8')
    
    return test_dir


def test_extractor_initialization(extractor):
    """Test TextExtractor initialization."""
    assert extractor is not None
    assert len(extractor.supported_formats) > 0


def test_extract_txt_file(extractor, test_files_dir):
    """Test extraction from TXT file."""
    txt_file = test_files_dir / "test.txt"
    result = extractor.extract(str(txt_file))
    
    assert result['text'] is not None
    assert "test text file" in result['text']
    assert result['metadata']['file_format'] == '.txt'
    assert result['metadata']['file_name'] == 'test.txt'


def test_extract_json_file(extractor, test_files_dir):
    """Test extraction from JSON file."""
    json_file = test_files_dir / "test.json"
    result = extractor.extract(str(json_file))
    
    assert result['text'] is not None
    assert result['raw_data'] is not None
    assert "Lost wallet" in result['text']
    assert result['raw_data']['title'] == "Lost wallet"
    assert result['metadata']['file_format'] == '.json'


def test_extract_html_file(extractor, test_files_dir):
    """Test extraction from HTML file."""
    html_file = test_files_dir / "test.html"
    result = extractor.extract(str(html_file))
    
    assert result['text'] is not None
    assert "Lost Items" in result['text']
    assert "test HTML document" in result['text']
    assert result['metadata']['file_format'] == '.html'


def test_extract_nonexistent_file(extractor):
    """Test extraction from nonexistent file."""
    with pytest.raises(FileNotFoundError):
        extractor.extract("nonexistent_file.txt")


def test_extract_unsupported_format(extractor, tmp_path):
    """Test extraction from unsupported format."""
    unsupported_file = tmp_path / "test.exe"
    unsupported_file.write_bytes(b"binary data")
    
    with pytest.raises(ValueError):
        extractor.extract(str(unsupported_file))


def test_convenience_function(test_files_dir):
    """Test convenience function."""
    txt_file = test_files_dir / "test.txt"
    result = extract_from_file(str(txt_file))
    
    assert result is not None
    assert 'text' in result
    assert 'metadata' in result
