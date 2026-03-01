"""
Face Recognition Service using DeepFace
"""

import numpy as np
import os
from PIL import Image
from typing import Tuple, Optional
import cv2
from deepface import DeepFace
import tensorflow as tf

from config import settings


class FaceRecognitionService:
    """Handle face detection and vectorization using DeepFace"""
    
    def __init__(self):
        self.model_name = settings.DEEPFACE_MODEL
        self.backend = settings.DEEPFACE_BACKEND
        self.detector = settings.DEEPFACE_DETECTOR
        
        # Suppress TensorFlow warnings
        tf.get_logger().setLevel('ERROR')
        os.environ['TF_CPP_MIN_LOG_LEVEL'] = '2'
    
    async def extract_vector(self, image: Image.Image) -> Tuple[Optional[np.ndarray], float]:
        """
        Extract face vector from image using DeepFace.
        
        Args:
            image: PIL Image object
            
        Returns:
            Tuple of (vector, confidence_score) or (None, 0.0) if no face detected
        """
        try:
            # Convert PIL Image to numpy array
            img_array = np.array(image)
            
            # Ensure image is in RGB format
            if len(img_array.shape) == 2:  # Grayscale
                img_array = cv2.cvtColor(img_array, cv2.COLOR_GRAY2RGB)
            elif img_array.shape[2] == 4:  # RGBA
                img_array = cv2.cvtColor(img_array, cv2.COLOR_RGBA2RGB)
            
            # Extract embedding using DeepFace
            embedding = DeepFace.represent(
                img_path=img_array,
                model_name=self.model_name,
                detector_backend=self.detector,
                enforce_detection=True,
                align=True
            )
            
            if not embedding:
                return None, 0.0
            
            # DeepFace returns a list of dictionaries
            # Each dict has 'embedding' key with the vector
            vector = np.array(embedding[0]['embedding'], dtype=np.float32)
            
            # Calculate confidence (normalize to 0-1)
            # DeepFace doesn't provide explicit confidence, so we use a heuristic
            confidence = min(1.0, np.linalg.norm(vector) / 10.0)
            
            return vector, float(confidence)
            
        except ValueError as e:
            # No face detected
            if "Face could not be detected" in str(e):
                return None, 0.0
            raise
        except Exception as e:
            raise Exception(f"Face extraction failed: {str(e)}")
    
    async def detect_liveness(
        self,
        image: Image.Image,
        liveness_type: str = "passive"
    ) -> Tuple[bool, float, float]:
        """
        Detect if face is live (real person vs photo/spoof).
        
        Args:
            image: PIL Image object
            liveness_type: 'active' (requires movement) or 'passive' (texture analysis)
            
        Returns:
            Tuple of (is_live, liveness_score, confidence)
        """
        try:
            # Convert PIL Image to numpy array
            img_array = np.array(image)
            
            if len(img_array.shape) == 2:
                img_array = cv2.cvtColor(img_array, cv2.COLOR_GRAY2RGB)
            elif img_array.shape[2] == 4:
                img_array = cv2.cvtColor(img_array, cv2.COLOR_RGBA2RGB)
            
            if liveness_type == "passive":
                # Passive liveness: texture analysis using DeepFace
                # This is a simplified implementation
                # In production, use a dedicated liveness detection model
                
                # Use DeepFace's built-in liveness (if available)
                # For now, we'll use a heuristic based on image quality
                gray = cv2.cvtColor(img_array, cv2.COLOR_RGB2GRAY)
                
                # Calculate Laplacian variance (blur detection)
                laplacian_var = cv2.Laplacian(gray, cv2.CV_64F).var()
                
                # Calculate edge density
                edges = cv2.Canny(gray, 50, 150)
                edge_density = np.sum(edges > 0) / (edges.shape[0] * edges.shape[1])
                
                # Heuristic: real faces have higher edge density and variance
                liveness_score = min(1.0, (laplacian_var / 100.0 + edge_density) / 2.0)
                is_live = liveness_score >= settings.PASSIVE_LIVENESS_THRESHOLD
                
                return is_live, float(liveness_score), float(liveness_score)
            
            elif liveness_type == "active":
                # Active liveness: requires user movement
                # This would typically require multiple frames
                # For single image, we use a stricter threshold
                
                # Similar analysis but with higher threshold
                gray = cv2.cvtColor(img_array, cv2.COLOR_RGB2GRAY)
                laplacian_var = cv2.Laplacian(gray, cv2.CV_64F).var()
                edges = cv2.Canny(gray, 50, 150)
                edge_density = np.sum(edges > 0) / (edges.shape[0] * edges.shape[1])
                
                liveness_score = min(1.0, (laplacian_var / 100.0 + edge_density) / 2.0)
                is_live = liveness_score >= settings.ACTIVE_LIVENESS_THRESHOLD
                
                return is_live, float(liveness_score), float(liveness_score)
            
            else:
                raise ValueError(f"Unknown liveness type: {liveness_type}")
                
        except Exception as e:
            # On error, assume not live (fail secure)
            return False, 0.0, 0.0
    
    async def detect_face(self, image: Image.Image) -> Optional[Tuple[int, int, int, int]]:
        """
        Detect face bounding box in image.
        
        Returns:
            Bounding box as (x, y, width, height) or None
        """
        try:
            img_array = np.array(image)
            
            if len(img_array.shape) == 2:
                img_array = cv2.cvtColor(img_array, cv2.COLOR_GRAY2RGB)
            elif img_array.shape[2] == 4:
                img_array = cv2.cvtColor(img_array, cv2.COLOR_RGBA2RGB)
            
            # Use DeepFace to detect face
            face_objs = DeepFace.extract_faces(
                img_path=img_array,
                detector_backend=self.detector,
                enforce_detection=False
            )
            
            if not face_objs or len(face_objs) == 0:
                return None
            
            # Return first face bounding box
            # Note: DeepFace doesn't directly return bbox, so we use OpenCV
            face_cascade = cv2.CascadeClassifier(cv2.data.haarcascades + 'haarcascade_frontalface_default.xml')
            gray = cv2.cvtColor(img_array, cv2.COLOR_RGB2GRAY)
            faces = face_cascade.detectMultiScale(gray, 1.1, 4)
            
            if len(faces) > 0:
                x, y, w, h = faces[0]
                return (int(x), int(y), int(w), int(h))
            
            return None
            
        except Exception as e:
            return None

