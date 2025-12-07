"""
NLP-based converter that transforms extracted text into lost-items schema.
Uses natural language processing to identify and extract relevant information.
"""

import json
import re
from datetime import datetime
from typing import Dict, List, Optional, Any
import logging

# NLP libraries
try:
    import spacy
    SPACY_AVAILABLE = True
except ImportError:
    SPACY_AVAILABLE = False

try:
    from transformers import pipeline
    TRANSFORMERS_AVAILABLE = True
except ImportError:
    TRANSFORMERS_AVAILABLE = False

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class NLPConverter:
    """Convert extracted text to lost-items schema using NLP."""
    
    # Common categories for lost items
    CATEGORIES = [
        "dokumenty", "elektronika", "biżuteria", "odzież", "torby i plecaki",
        "klucze", "portfele", "telefony", "zwierzęta", "pojazdy", "inne"
    ]
    
    # Keywords for category detection
    CATEGORY_KEYWORDS = {
        "dokumenty": ["dowód", "paszport", "prawo jazdy", "dokument", "legitymacja", "karta"],
        "elektronika": ["telefon", "laptop", "tablet", "komputer", "słuchawki", "smartwatch", "aparat"],
        "biżuteria": ["pierścionek", "naszyjnik", "bransoleta", "zegarek", "kolczyki", "biżuteria"],
        "odzież": ["kurtka", "płaszcz", "sweter", "spodnie", "buty", "czapka", "szalik", "rękawiczki"],
        "torby i plecaki": ["torba", "plecak", "torebka", "walizka", "saszetka"],
        "klucze": ["klucz", "kluczyk", "brelok"],
        "portfele": ["portfel", "portmonetka", "etui"],
        "telefony": ["telefon", "smartfon", "iphone", "samsung", "komórka"],
        "zwierzęta": ["pies", "kot", "zwierzę", "pupil"],
        "pojazdy": ["rower", "hulajnoga", "samochód", "motor", "pojazd"]
    }
    
    # Location patterns
    LOCATION_PATTERNS = [
        r"znaleziono\s+(?:w|na|przy)\s+([^,\.]+)",
        r"miejsce\s*:\s*([^,\.\n]+)",
        r"lokalizacja\s*:\s*([^,\.\n]+)",
        r"adres\s*:\s*([^,\.\n]+)",
        r"(?:w|na|przy)\s+(ul\.|ulicy|ulica|placu|placu)\s+([^,\.\n]+)",
    ]
    
    # Date patterns
    DATE_PATTERNS = [
        r"(\d{1,2})[\./-](\d{1,2})[\./-](\d{4})",  # DD.MM.YYYY or DD/MM/YYYY
        r"(\d{4})-(\d{1,2})-(\d{1,2})",  # YYYY-MM-DD
        r"(\d{1,2})\s+(stycznia|lutego|marca|kwietnia|maja|czerwca|lipca|sierpnia|września|października|listopada|grudnia)\s+(\d{4})",
    ]
    
    MONTHS_PL = {
        "stycznia": 1, "lutego": 2, "marca": 3, "kwietnia": 4,
        "maja": 5, "czerwca": 6, "lipca": 7, "sierpnia": 8,
        "września": 9, "października": 10, "listopada": 11, "grudnia": 12
    }
    
    # Contact patterns
    EMAIL_PATTERN = r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b'
    PHONE_PATTERN = r'(?:\+48\s?)?(?:\d{3}[\s-]?\d{3}[\s-]?\d{3}|\d{2}[\s-]?\d{3}[\s-]?\d{2}[\s-]?\d{2})'
    
    def __init__(self, use_spacy: bool = False, use_transformers: bool = False):
        """
        Initialize NLP converter.
        
        Args:
            use_spacy: Use spaCy for advanced NLP (requires model installation)
            use_transformers: Use transformers for zero-shot classification
        """
        self.nlp = None
        self.classifier = None
        
        if use_spacy and SPACY_AVAILABLE:
            try:
                # Try to load Polish model
                self.nlp = spacy.load("pl_core_news_sm")
                logger.info("Loaded spaCy Polish model")
            except OSError:
                logger.warning("spaCy Polish model not found. Install with: python -m spacy download pl_core_news_sm")
        
        if use_transformers and TRANSFORMERS_AVAILABLE:
            try:
                # Zero-shot classification for categories
                self.classifier = pipeline("zero-shot-classification", model="facebook/bart-large-mnli")
                logger.info("Loaded transformers classifier")
            except Exception as e:
                logger.warning(f"Could not load transformers: {e}")
    
    def convert(self, extracted_data: Dict[str, Any], dataset_id: Optional[str] = None) -> Dict[str, Any]:
        """
        Convert extracted text to lost-items schema.
        
        Args:
            extracted_data: Output from text_extractor
            dataset_id: Optional dataset ID to associate the item with
            
        Returns:
            Dict matching LostItem schema
        """
        text = extracted_data.get("text", "")
        raw_data = extracted_data.get("raw_data")
        metadata = extracted_data.get("metadata", {})
        
        # Try structured data first (CSV, JSON, XML)
        if raw_data:
            item = self._convert_structured_data(raw_data, text)
        else:
            item = self._convert_unstructured_text(text)
        
        # Add metadata
        item["metadata"] = {
            "source_file": metadata.get("file_name"),
            "source_format": metadata.get("file_format"),
            "conversion_timestamp": datetime.utcnow().isoformat(),
        }
        
        if dataset_id:
            item["dataset_id"] = dataset_id
        
        return item
    
    def _convert_structured_data(self, raw_data: Any, text: str) -> Dict[str, Any]:
        """Convert structured data (CSV, JSON, XML) to lost-items schema."""
        item = self._create_empty_item()
        
        # Handle CSV rows
        if isinstance(raw_data, list) and len(raw_data) > 0:
            # Take first row for now (can be extended to handle multiple items)
            row = raw_data[0]
            item = self._map_structured_fields(row, item)
        
        # Handle JSON/XML objects
        elif isinstance(raw_data, dict):
            item = self._map_structured_fields(raw_data, item)
        
        # Fallback to text extraction for missing fields
        item = self._fill_missing_fields(item, text)
        
        return item
    
    def _map_structured_fields(self, data: Dict, item: Dict) -> Dict:
        """Map structured data fields to lost-items schema."""
        
        # Common field name mappings
        field_mappings = {
            "title": ["title", "tytuł", "nazwa", "przedmiot", "name", "item"],
            "description": ["description", "opis", "desc", "details", "szczegóły"],
            "category": ["category", "kategoria", "typ", "type"],
            "location": ["location", "lokalizacja", "miejsce", "adres", "address", "place"],
            "found_date": ["found_date", "data_znalezienia", "date", "data", "found"],
            "reporting_date": ["reporting_date", "data_zgłoszenia", "report_date"],
            "reporting_location": ["reporting_location", "miejsce_zgłoszenia"],
            "contact_email": ["email", "e-mail", "contact_email", "kontakt"],
            "contact_phone": ["phone", "telefon", "tel", "contact_phone", "phone_number"],
            "status": ["status", "stan"],
        }
        
        # Try to map fields
        for target_field, possible_names in field_mappings.items():
            for name in possible_names:
                value = self._find_field_value(data, name)
                if value:
                    if target_field in ["found_date", "reporting_date"]:
                        item[target_field] = self._parse_date(str(value))
                    else:
                        item[target_field] = str(value).strip()
                    break
        
        return item
    
    def _find_field_value(self, data: Dict, field_name: str) -> Optional[str]:
        """Find field value in nested dict (case-insensitive)."""
        field_lower = field_name.lower()
        
        for key, value in data.items():
            if key.lower() == field_lower:
                if isinstance(value, (str, int, float)):
                    return str(value)
                elif isinstance(value, dict) and '_text' in value:
                    return value['_text']
        
        return None
    
    def _convert_unstructured_text(self, text: str) -> Dict[str, Any]:
        """Convert unstructured text to lost-items schema using NLP."""
        item = self._create_empty_item()
        
        # Extract title (first significant line)
        item["title"] = self._extract_title(text)
        
        # Extract description (use full text as fallback)
        item["description"] = self._extract_description(text)
        
        # Extract category
        item["category"] = self._extract_category(text)
        
        # Extract location
        item["location"] = self._extract_location(text)
        
        # Extract dates
        dates = self._extract_dates(text)
        if dates:
            item["found_date"] = dates[0]
            if len(dates) > 1:
                item["reporting_date"] = dates[1]
        
        # Extract contact info
        item["contact_email"] = self._extract_email(text)
        item["contact_phone"] = self._extract_phone(text)
        
        return item
    
    def _create_empty_item(self) -> Dict[str, Any]:
        """Create empty item with default values."""
        return {
            "title": "",
            "description": "",
            "category": "inne",
            "location": "",
            "found_date": None,
            "reporting_date": None,
            "reporting_location": "",
            "contact_email": "",
            "contact_phone": "",
            "status": "pending",
        }
    
    def _fill_missing_fields(self, item: Dict, text: str) -> Dict:
        """Fill in missing fields from text extraction."""
        if not item.get("title"):
            item["title"] = self._extract_title(text)
        
        if not item.get("description"):
            item["description"] = self._extract_description(text)
        
        if not item.get("category") or item["category"] == "inne":
            item["category"] = self._extract_category(text)
        
        if not item.get("location"):
            item["location"] = self._extract_location(text)
        
        if not item.get("found_date"):
            dates = self._extract_dates(text)
            if dates:
                item["found_date"] = dates[0]
        
        if not item.get("contact_email"):
            item["contact_email"] = self._extract_email(text)
        
        if not item.get("contact_phone"):
            item["contact_phone"] = self._extract_phone(text)
        
        return item
    
    def _extract_title(self, text: str) -> str:
        """Extract title from text."""
        lines = [line.strip() for line in text.split('\n') if line.strip()]
        
        # Take first significant line (longer than 10 chars)
        for line in lines:
            if len(line) > 10 and not line.startswith('['):
                return line[:200]  # Limit length
        
        return "Nieznany przedmiot"
    
    def _extract_description(self, text: str) -> str:
        """Extract description from text."""
        # Clean up text
        text = re.sub(r'\s+', ' ', text).strip()
        
        # Limit length
        max_length = 1000
        if len(text) > max_length:
            return text[:max_length] + "..."
        
        return text
    
    def _extract_category(self, text: str) -> str:
        """Extract category from text using keyword matching."""
        text_lower = text.lower()
        
        # Score each category
        scores = {}
        for category, keywords in self.CATEGORY_KEYWORDS.items():
            score = sum(1 for keyword in keywords if keyword in text_lower)
            if score > 0:
                scores[category] = score
        
        # Return category with highest score
        if scores:
            return max(scores, key=scores.get)
        
        # Try transformers classifier if available
        if self.classifier:
            try:
                result = self.classifier(text[:512], self.CATEGORIES)
                if result['scores'][0] > 0.5:
                    return result['labels'][0]
            except Exception as e:
                logger.warning(f"Classifier error: {e}")
        
        return "inne"
    
    def _extract_location(self, text: str) -> str:
        """Extract location from text."""
        for pattern in self.LOCATION_PATTERNS:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                location = match.group(1).strip()
                # Clean up location
                location = re.sub(r'\s+', ' ', location)
                return location[:200]
        
        # Use spaCy for location extraction if available
        if self.nlp:
            try:
                doc = self.nlp(text[:1000])  # Limit text length
                locations = [ent.text for ent in doc.ents if ent.label_ == "GPE"]
                if locations:
                    return locations[0]
            except Exception as e:
                logger.warning(f"spaCy error: {e}")
        
        return ""
    
    def _extract_dates(self, text: str) -> List[Optional[str]]:
        """Extract dates from text."""
        dates = []
        
        for pattern in self.DATE_PATTERNS:
            matches = re.finditer(pattern, text, re.IGNORECASE)
            for match in matches:
                date_str = self._parse_date(match.group(0))
                if date_str:
                    dates.append(date_str)
        
        return dates
    
    def _parse_date(self, date_str: str) -> Optional[str]:
        """Parse date string to ISO format."""
        if not date_str:
            return None
        
        # Try common date formats
        formats = [
            "%d.%m.%Y", "%d/%m/%Y", "%Y-%m-%d",
            "%d-%m-%Y", "%Y/%m/%d"
        ]
        
        for fmt in formats:
            try:
                dt = datetime.strptime(date_str.strip(), fmt)
                return dt.isoformat()
            except ValueError:
                continue
        
        # Try parsing Polish month names
        match = re.search(r'(\d{1,2})\s+(stycznia|lutego|marca|kwietnia|maja|czerwca|lipca|sierpnia|września|października|listopada|grudnia)\s+(\d{4})',
                         date_str, re.IGNORECASE)
        if match:
            day = int(match.group(1))
            month = self.MONTHS_PL[match.group(2).lower()]
            year = int(match.group(3))
            try:
                dt = datetime(year, month, day)
                return dt.isoformat()
            except ValueError:
                pass
        
        return None
    
    def _extract_email(self, text: str) -> str:
        """Extract email from text."""
        match = re.search(self.EMAIL_PATTERN, text)
        return match.group(0) if match else ""
    
    def _extract_phone(self, text: str) -> str:
        """Extract phone number from text."""
        match = re.search(self.PHONE_PATTERN, text)
        if match:
            # Clean up phone number
            phone = re.sub(r'[\s-]', '', match.group(0))
            return phone
        return ""
    
    def convert_batch(self, extracted_items: List[Dict[str, Any]], dataset_id: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Convert multiple extracted items to lost-items schema.
        
        Args:
            extracted_items: List of outputs from text_extractor
            dataset_id: Optional dataset ID to associate items with
            
        Returns:
            List of dicts matching LostItem schema
        """
        items = []
        for i, extracted_data in enumerate(extracted_items):
            try:
                item = self.convert(extracted_data, dataset_id)
                items.append(item)
                logger.info(f"Converted item {i+1}/{len(extracted_items)}")
            except Exception as e:
                logger.error(f"Error converting item {i+1}: {e}")
        
        return items


def convert_text_to_item(extracted_data: Dict[str, Any], dataset_id: Optional[str] = None) -> Dict[str, Any]:
    """
    Convenience function to convert extracted text to lost item.
    
    Args:
        extracted_data: Output from text_extractor
        dataset_id: Optional dataset ID
        
    Returns:
        Dict matching LostItem schema
    """
    converter = NLPConverter()
    return converter.convert(extracted_data, dataset_id)


if __name__ == "__main__":
    import sys
    from text_extractor import extract_from_file
    
    if len(sys.argv) < 2:
        print("Usage: python nlp_converter.py <file_path>")
        sys.exit(1)
    
    file_path = sys.argv[1]
    
    try:
        # Extract text
        print("Extracting text...")
        extracted_data = extract_from_file(file_path)
        
        # Convert to lost item
        print("Converting to lost-items schema...")
        converter = NLPConverter()
        item = converter.convert(extracted_data)
        
        # Display result
        print(f"\n{'='*80}")
        print("CONVERTED LOST ITEM:")
        print(f"{'='*80}\n")
        print(json.dumps(item, indent=2, ensure_ascii=False))
        
    except Exception as e:
        logger.error(f"Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
