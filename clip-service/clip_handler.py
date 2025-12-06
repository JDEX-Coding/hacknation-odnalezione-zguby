"""
CLIP Model Handler
Handles loading and inference of the CLIP model for text and image embeddings.
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
    """Handler for CLIP model operations."""
    
    def __init__(self, model_name: str = "openai/clip-vit-base-patch32"):
        """
        Initialize CLIP handler.
        
        Args:
            model_name: Name of the CLIP model to use from HuggingFace
        """
        self.model_name = model_name
        self.model = None
        self.processor = None
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        logger.info(f"Using device: {self.device}")
    
    def load_model(self):
        """Load the CLIP model and processor."""
        try:
            logger.info(f"Loading CLIP model: {self.model_name}")
            self.processor = CLIPProcessor.from_pretrained(self.model_name)
            self.model = CLIPModel.from_pretrained(self.model_name)
            self.model.to(self.device)
            self.model.eval()
            logger.info("✅ CLIP model loaded successfully")
        except Exception as e:
            logger.error(f"Failed to load CLIP model: {e}")
            raise
    
    def encode_text(self, text: Union[str, List[str]]) -> np.ndarray:
        """
        Encode text into embeddings.
        
        Args:
            text: Single text string or list of text strings
            
        Returns:
            Normalized embedding vector(s)
        """
        if self.model is None or self.processor is None:
            raise RuntimeError("Model not loaded. Call load_model() first.")
        
        try:
            # Ensure text is a list
            if isinstance(text, str):
                text = [text]
            
            # Process text
            inputs = self.processor(
                text=text,
                return_tensors="pt",
                padding=True,
                truncation=True,
                max_length=77
            )
            
            # Move to device
            inputs = {k: v.to(self.device) for k, v in inputs.items()}
            
            # Generate embeddings
            with torch.no_grad():
                text_features = self.model.get_text_features(**inputs)
                
                # Normalize embeddings
                text_features = text_features / text_features.norm(dim=-1, keepdim=True)
            
            # Convert to numpy
            embeddings = text_features.cpu().numpy()
            
            # Return single embedding if single input
            if len(embeddings) == 1:
                return embeddings[0]
            return embeddings
            
        except Exception as e:
            logger.error(f"Error encoding text: {e}")
            raise
    
    def encode_image(self, image_path: Union[str, Path]) -> np.ndarray:
        """
        Encode image into embeddings.
        
        Args:
            image_path: Path to the image file
            
        Returns:
            Normalized embedding vector
        """
        if self.model is None or self.processor is None:
            raise RuntimeError("Model not loaded. Call load_model() first.")
        
        try:
            # Load image
            image = Image.open(image_path).convert('RGB')
            
            # Process image
            inputs = self.processor(
                images=image,
                return_tensors="pt"
            )
            
            # Move to device
            inputs = {k: v.to(self.device) for k, v in inputs.items()}
            
            # Generate embeddings
            with torch.no_grad():
                image_features = self.model.get_image_features(**inputs)
                
                # Normalize embeddings
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
            image_embeddings = np.array([e for e in image_embeddings if e is not None])
        
        return text_embeddings, image_embeddings
    
    def compute_similarity(
        self,
        text_embedding: np.ndarray,
        image_embedding: np.ndarray
    ) -> float:
        """
        Compute cosine similarity between text and image embeddings.
        
        Args:
            text_embedding: Text embedding vector
            image_embedding: Image embedding vector
            
        Returns:
            Similarity score (0-1)
        """
        similarity = np.dot(text_embedding, image_embedding)
        return float(similarity)
    
    def get_embedding_dimension(self) -> int:
        """
        Get the dimension of embeddings produced by the model.
        
        Returns:
            Embedding dimension
        """
        if self.model is None:
            raise RuntimeError("Model not loaded. Call load_model() first.")
        
        return self.model.config.projection_dim
