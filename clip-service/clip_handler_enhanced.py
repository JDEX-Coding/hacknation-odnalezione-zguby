"""
Enhanced CLIP Model Handler with Unified Embeddings
Fixes semantic search accuracy by using single model for text and images.
"""

import logging
from typing import List, Union, Optional
from pathlib import Path

import torch
import numpy as np
from PIL import Image
from transformers import CLIPProcessor, CLIPModel

logger = logging.getLogger(__name__)


class CLIPHandler:
    """Handler for CLIP model operations using unified model for text and images."""

    def __init__(self, model_name: str = "openai/clip-vit-base-patch32"):
        """
        Initialize CLIP handler with SINGLE unified model.

        Args:
            model_name: Name of the CLIP model to use from HuggingFace (for BOTH text and images)
        """
        self.model_name = model_name
        self.model = None
        self.processor = None
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        logger.info(f"Using device: {self.device}")
        logger.info(f"🔧 Using UNIFIED CLIP model: {model_name}")

    def load_model(self):
        """Load the CLIP model and processor."""
        try:
            logger.info(f"Loading unified CLIP model: {self.model_name}")
            self.processor = CLIPProcessor.from_pretrained(self.model_name)
            self.model = CLIPModel.from_pretrained(self.model_name)
            self.model.to(self.device)
            self.model.eval()

            # Log embedding dimension
            embedding_dim = self.model.config.projection_dim
            logger.info(f"✅ CLIP model loaded successfully")
            logger.info(f"📐 Embedding dimension: {embedding_dim}")

        except Exception as e:
            logger.error(f"Failed to load CLIP model: {e}")
            raise

    def encode_text(self, text: Union[str, List[str]]) -> np.ndarray:
        """
        Encode text into embeddings using CLIP text encoder.

        Args:
            text: Single text string or list of text strings

        Returns:
            Normalized embedding vector(s) - shape (512,) or (n, 512)
        """
        if self.model is None or self.processor is None:
            raise RuntimeError("Model not loaded. Call load_model() first.")

        try:
            # Convert single string to list for consistent processing
            if isinstance(text, str):
                text = [text]
                single_input = True
            else:
                single_input = False

            # Process text through CLIP processor
            inputs = self.processor(
                text=text,
                return_tensors="pt",
                padding=True,
                truncation=True,
                max_length=77  # CLIP's max text length
            )

            # Move to device
            inputs = {k: v.to(self.device) for k, v in inputs.items()}

            # Generate text embeddings
            with torch.no_grad():
                text_features = self.model.get_text_features(**inputs)

                # Normalize embeddings (critical for cosine similarity)
                text_features = text_features / text_features.norm(dim=-1, keepdim=True)

            # Convert to numpy
            embeddings = text_features.cpu().numpy()

            # Return single vector if single input
            if single_input:
                return embeddings[0]

            return embeddings

        except Exception as e:
            logger.error(f"Error encoding text: {e}")
            raise

    def encode_image(self, image_path: Union[str, Path]) -> np.ndarray:
        """
        Encode image into embeddings using CLIP image encoder.

        Args:
            image_path: Path to the image file

        Returns:
            Normalized embedding vector - shape (512,)
        """
        if self.model is None or self.processor is None:
            raise RuntimeError("Model not loaded. Call load_model() first.")

        try:
            # Load and convert image to RGB
            image = Image.open(image_path).convert('RGB')

            # Process image through CLIP processor
            inputs = self.processor(
                images=image,
                return_tensors="pt"
            )

            # Move to device
            inputs = {k: v.to(self.device) for k, v in inputs.items()}

            # Generate image embeddings
            with torch.no_grad():
                image_features = self.model.get_image_features(**inputs)

                # Normalize embeddings (critical for cosine similarity)
                image_features = image_features / image_features.norm(dim=-1, keepdim=True)

            # Convert to numpy
            embedding = image_features.cpu().numpy()[0]

            return embedding

        except Exception as e:
            logger.error(f"Error encoding image {image_path}: {e}")
            raise

    def encode_batch(
        self,
        texts: Optional[List[str]] = None,
        image_paths: Optional[List[Union[str, Path]]] = None
    ) -> tuple:
        """
        Encode a batch of texts and/or images.

        Args:
            texts: List of text strings
            image_paths: List of image file paths

        Returns:
            Tuple of (text_embeddings, image_embeddings)
        """
        text_embeddings = None
        image_embeddings = None

        if texts:
            text_embeddings = self.encode_text(texts)

        if image_paths:
            image_embeddings = []
            for image_path in image_paths:
                try:
                    embedding = self.encode_image(image_path)
                    image_embeddings.append(embedding)
                except Exception as e:
                    logger.error(f"Failed to encode image {image_path}: {e}")
                    image_embeddings.append(None)

            # Filter out None values and convert to array
            valid_embeddings = [e for e in image_embeddings if e is not None]
            if valid_embeddings:
                image_embeddings = np.array(valid_embeddings)
            else:
                image_embeddings = None

        return text_embeddings, image_embeddings

    def compute_similarity(
        self,
        text_embedding: np.ndarray,
        image_embedding: np.ndarray
    ) -> float:
        """
        Compute cosine similarity between text and image embeddings.
        Now meaningful because both embeddings are from the same CLIP model!

        Args:
            text_embedding: Text embedding vector
            image_embedding: Image embedding vector

        Returns:
            Similarity score (0-1)
        """
        similarity = np.dot(text_embedding, image_embedding)
        return float(similarity)

    def combine_embeddings(
        self,
        text_embedding: np.ndarray,
        image_embedding: np.ndarray,
        text_weight: float = 0.5
    ) -> np.ndarray:
        """
        Combine text and image embeddings with weighted average.

        Since both embeddings are from the same CLIP model, they share
        the same semantic space and can be meaningfully combined.

        Args:
            text_embedding: Text embedding vector (512,)
            image_embedding: Image embedding vector (512,)
            text_weight: Weight for text embedding (0-1), image gets (1-text_weight)

        Returns:
            Combined normalized embedding vector (512,)
        """
        if not 0 <= text_weight <= 1:
            raise ValueError("text_weight must be between 0 and 1")

        image_weight = 1.0 - text_weight

        # Weighted combination
        combined = text_weight * text_embedding + image_weight * image_embedding

        # Re-normalize (important!)
        combined = combined / np.linalg.norm(combined)

        return combined

    def get_embedding_dimension(self) -> int:
        """
        Get the dimension of embeddings produced by the model.

        Returns:
            Embedding dimension (should be 512 for clip-vit-base-patch32)
        """
        if self.model is None:
            raise RuntimeError("Model not loaded. Call load_model() first.")

        return self.model.config.projection_dim


class TextPreprocessor:
    """Enhanced text preprocessing for Polish lost items."""

    # Polish stopwords (common words to filter)
    POLISH_STOPWORDS = {
        'i', 'w', 'z', 'na', 'do', 'po', 'dla', 'od', 'ze', 'o', 'przy',
        'przez', 'pod', 'nad', 'za', 'przed', 'bez', 'u', 'we', 'został',
        'została', 'było', 'jest', 'są', 'będzie', 'ten', 'ta', 'to', 'te'
    }

    def normalize_text(self, text: str) -> str:
        """
        Normalize Polish text for better embedding quality.

        Args:
            text: Input text

        Returns:
            Normalized text
        """
        import re

        # Convert to lowercase
        text = text.lower()

        # Remove extra whitespace
        text = re.sub(r'\s+', ' ', text).strip()

        return text

    def extract_keywords(self, text: str) -> List[str]:
        """
        Extract important keywords from text.

        Args:
            text: Input text

        Returns:
            List of keywords
        """
        words = self.normalize_text(text).split()
        keywords = [
            w for w in words
            if w not in self.POLISH_STOPWORDS and len(w) > 2
        ]
        return keywords

    def create_enhanced_text(
        self,
        title: str,
        description: str,
        category: str,
        location: str = ""
    ) -> str:
        """
        Create enhanced search text with optimal structure.

        Args:
            title: Item title
            description: Item description
            category: Item category
            location: Item location

        Returns:
            Enhanced text optimized for CLIP embedding
        """
        parts = []

        # Emphasize title by repeating it
        if title:
            parts.append(f"{title} {title}")

        # Add full description
        if description:
            parts.append(description)

        # Add category with Polish label for better context
        if category:
            parts.append(f"kategoria: {category}")

        # Add location
        if location:
            parts.append(f"miejsce: {location}")

        # Join and normalize
        enhanced_text = " ".join(parts)
        return self.normalize_text(enhanced_text)


class QueryEnhancer:
    """Enhance user queries for better search results."""

    # Polish to English translations for common lost items
    TRANSLATIONS = {
        'portfel': 'wallet',
        'portmonetka': 'wallet purse',
        'telefon': 'phone mobile smartphone',
        'komórka': 'phone mobile',
        'klucze': 'keys',
        'kluczyki': 'keys',
        'torebka': 'bag purse handbag',
        'torba': 'bag',
        'plecak': 'backpack rucksack',
        'parasolka': 'umbrella',
        'parasol': 'umbrella',
        'okulary': 'glasses eyeglasses',
        'zegarek': 'watch',
        'laptop': 'laptop computer notebook',
        'czarny': 'black',
        'biały': 'white',
        'niebieski': 'blue',
        'czerwony': 'red',
        'zielony': 'green',
        'żółty': 'yellow',
        'szary': 'gray grey',
        'brązowy': 'brown',
    }

    def enhance_query(self, query: str) -> str:
        """
        Enhance query with translations and context.

        Args:
            query: User's search query

        Returns:
            Enhanced query with additional context
        """
        query = query.lower().strip()
        words = query.split()

        enhanced_parts = [query]  # Keep original query

        # Add English translations for known words
        for word in words:
            if word in self.TRANSLATIONS:
                enhanced_parts.append(self.TRANSLATIONS[word])

        # Join all parts
        enhanced_query = " ".join(enhanced_parts)

        return enhanced_query

    def expand_query(self, query: str) -> str:
        """
        Expand query with common patterns.

        Args:
            query: User's search query

        Returns:
            Expanded query
        """
        # Add common modifiers
        expanded = f"{query} znaleziony zgubiony lost found"
        return expanded
