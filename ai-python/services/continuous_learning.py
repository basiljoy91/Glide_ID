"""
Continuous Learning Service for Biometric Drift Correction
"""

import numpy as np
from typing import List, Optional
from datetime import datetime, timedelta

from config import settings
from services.database import DatabaseService


class ContinuousLearningService:
    """Handle continuous learning to update face vectors over time"""
    
    def __init__(self, database: Optional[DatabaseService] = None):
        # Reuse app-level pooled database service; avoid creating an unconnected instance.
        self.database = database or DatabaseService()
        self.learning_rate = settings.CONTINUOUS_LEARNING_RATE
        self.confidence_threshold = settings.CONTINUOUS_LEARNING_THRESHOLD
        self.max_frequency_days = settings.MAX_LEARNING_FREQUENCY_DAYS
    
    async def can_update(self, user_id: str, tenant_id: str) -> bool:
        """
        Check if continuous learning update is allowed (throttling).
        Max once per week per user.
        
        Returns:
            True if update is allowed, False if throttled
        """
        try:
            last_update = await self.database.get_last_learning_update(user_id, tenant_id)
            
            if last_update is None:
                return True  # Never updated, allow
            
            # Check if enough time has passed
            time_since_update = datetime.utcnow() - last_update
            days_since_update = time_since_update.days
            
            return days_since_update >= self.max_frequency_days
            
        except Exception:
            # On error, allow update (fail open for availability)
            return True
    
    async def update_vector(
        self,
        existing_vector: List[float] | np.ndarray,
        new_vector: List[float] | np.ndarray,
        confidence: float
    ) -> Optional[List[float]]:
        """
        Update existing vector by blending with new vector.
        Only updates if confidence meets threshold (98%+).
        
        Formula: updated = (1 - learning_rate) * existing + learning_rate * new
        Default: 95% old, 5% new
        
        Args:
            existing_vector: Current stored vector
            new_vector: New vector from recent match
            confidence: Confidence score of the match
            
        Returns:
            Updated vector or None if update not applied
        """
        # Check confidence threshold
        if confidence < self.confidence_threshold:
            return None
        
        # Convert to numpy arrays
        existing = np.array(existing_vector, dtype=np.float32)
        new = np.array(new_vector, dtype=np.float32)
        
        # Ensure same dimension
        if existing.shape != new.shape:
            return None
        
        # Blend vectors: 95% old, 5% new (configurable)
        updated = (1 - self.learning_rate) * existing + self.learning_rate * new
        
        # Normalize to maintain unit vector properties
        updated = updated / (np.linalg.norm(updated) + 1e-8)
        
        return updated.tolist()
    
    async def should_update(
        self,
        similarity: float,
        last_update: Optional[datetime]
    ) -> bool:
        """
        Determine if vector should be updated based on similarity and timing.
        
        Args:
            similarity: Similarity score from match
            last_update: Last learning update timestamp
            
        Returns:
            True if update should be applied
        """
        # Check confidence threshold
        if similarity < self.confidence_threshold:
            return False
        
        # Check timing
        if last_update is None:
            return True
        
        time_since_update = datetime.utcnow() - last_update
        days_since_update = time_since_update.days
        
        return days_since_update >= self.max_frequency_days
