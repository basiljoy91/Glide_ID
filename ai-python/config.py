"""
Configuration settings for AI Microservice
"""

from pydantic_settings import BaseSettings
from typing import List
import os


class Settings(BaseSettings):
    # Server Configuration
    HOST: str = os.getenv("HOST", "0.0.0.0")
    PORT: int = int(os.getenv("PORT", "8000"))
    DEBUG: bool = os.getenv("DEBUG", "false").lower() == "true"
    
    # API Security
    API_KEY: str = os.getenv("API_KEY", "change-me-in-production")
    
    # CORS
    CORS_ORIGINS: List[str] = [
        "http://localhost:3000",
        "http://localhost:3001",
        "https://*.vercel.app",
        "https://*.onrender.com",
    ]
    
    # Database Configuration (Supabase PostgreSQL)
    DATABASE_URL: str = os.getenv(
        "DATABASE_URL",
        "postgresql://user:password@localhost:5432/dbname"
    )
    DB_POOL_SIZE: int = int(os.getenv("DB_POOL_SIZE", "10"))
    DB_MAX_OVERFLOW: int = int(os.getenv("DB_MAX_OVERFLOW", "20"))
    
    # Encryption
    ENCRYPTION_KEY: str = os.getenv(
        "ENCRYPTION_KEY",
        "change-me-to-32-byte-key-in-production"
    )
    
    # DeepFace Configuration
    DEEPFACE_BACKEND: str = os.getenv("DEEPFACE_BACKEND", "opencv")
    DEEPFACE_MODEL: str = os.getenv("DEEPFACE_MODEL", "ArcFace")
    DEEPFACE_DETECTOR: str = os.getenv("DEEPFACE_DETECTOR", "opencv")
    
    # Vector Comparison Thresholds
    DEFAULT_SIMILARITY_THRESHOLD: float = float(os.getenv("DEFAULT_SIMILARITY_THRESHOLD", "0.62"))
    CONTINUOUS_LEARNING_THRESHOLD: float = float(os.getenv("CONTINUOUS_LEARNING_THRESHOLD", "0.98"))
    CONTINUOUS_LEARNING_RATE: float = float(os.getenv("CONTINUOUS_LEARNING_RATE", "0.05"))
    MAX_LEARNING_FREQUENCY_DAYS: int = int(os.getenv("MAX_LEARNING_FREQUENCY_DAYS", "7"))
    
    # Liveness Detection
    PASSIVE_LIVENESS_THRESHOLD: float = float(os.getenv("PASSIVE_LIVENESS_THRESHOLD", "0.62"))
    ACTIVE_LIVENESS_THRESHOLD: float = float(os.getenv("ACTIVE_LIVENESS_THRESHOLD", "0.72"))
    
    # Performance
    VECTOR_DIMENSION: int = int(os.getenv("VECTOR_DIMENSION", "512"))
    MAX_IMAGE_SIZE_MB: int = int(os.getenv("MAX_IMAGE_SIZE_MB", "10"))
    AI_WARMUP_ON_STARTUP: str = os.getenv("AI_WARMUP_ON_STARTUP", "true")

    @property
    def ai_warmup_on_startup(self) -> bool:
        normalized = str(self.AI_WARMUP_ON_STARTUP).strip().lower()
        if normalized in {"true", "1", "yes", "y", "on", "ture"}:
            return True
        if normalized in {"false", "0", "no", "n", "off"}:
            return False
        return True
    
    class Config:
        env_file = ".env"
        case_sensitive = True


settings = Settings()
