"""
Vector Comparison Service for Face Matching
"""

import numpy as np
from typing import List, Tuple
from sklearn.metrics.pairwise import cosine_similarity

from config import settings


class VectorComparisonService:
    """Handle vector similarity calculations for face matching"""
    
    async def cosine_similarity(
        self,
        vector1: List[float] | np.ndarray,
        vector2: List[float] | np.ndarray
    ) -> float:
        """
        Calculate cosine similarity between two face vectors.
        
        Args:
            vector1: First face vector
            vector2: Second face vector
            
        Returns:
            Similarity score between 0.0 and 1.0
        """
        # Convert to numpy arrays
        v1 = np.array(vector1, dtype=np.float32).reshape(1, -1)
        v2 = np.array(vector2, dtype=np.float32).reshape(1, -1)
        
        # Normalize vectors
        v1_norm = v1 / (np.linalg.norm(v1) + 1e-8)
        v2_norm = v2 / (np.linalg.norm(v2) + 1e-8)
        
        # Calculate cosine similarity
        similarity = np.dot(v1_norm, v2_norm.T)[0][0]
        
        # Ensure result is between 0 and 1
        similarity = max(0.0, min(1.0, similarity))
        
        return float(similarity)
    
    async def euclidean_distance(
        self,
        vector1: List[float] | np.ndarray,
        vector2: List[float] | np.ndarray
    ) -> float:
        """
        Calculate Euclidean distance between two vectors.
        Lower distance = higher similarity.
        
        Returns:
            Distance (lower is better)
        """
        v1 = np.array(vector1, dtype=np.float32)
        v2 = np.array(vector2, dtype=np.float32)
        
        distance = np.linalg.norm(v1 - v2)
        
        return float(distance)
    
    async def find_best_match(
        self,
        query_vector: List[float] | np.ndarray,
        candidate_vectors: List[Tuple[str, List[float]]],
        threshold: float = None
    ) -> List[Tuple[str, float]]:
        """
        Find best matching vectors from a list of candidates.
        
        Args:
            query_vector: Vector to match against
            candidate_vectors: List of (id, vector) tuples
            threshold: Minimum similarity threshold
            
        Returns:
            List of (id, similarity) tuples sorted by similarity (highest first)
        """
        if threshold is None:
            threshold = settings.DEFAULT_SIMILARITY_THRESHOLD
        
        matches = []
        
        for candidate_id, candidate_vector in candidate_vectors:
            similarity = await self.cosine_similarity(query_vector, candidate_vector)
            
            if similarity >= threshold:
                matches.append((candidate_id, similarity))
        
        # Sort by similarity (highest first)
        matches.sort(key=lambda x: x[1], reverse=True)
        
        return matches

