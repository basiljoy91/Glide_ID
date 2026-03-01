"""
AES-256 Encryption Utilities for Face Vectors
"""

from cryptography.fernet import Fernet
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.backends import default_backend
import base64
import numpy as np
from typing import List
import os

from config import settings


class EncryptionUtils:
    """Handle AES-256 encryption/decryption of face vectors"""
    
    def __init__(self):
        # Derive key from ENCRYPTION_KEY
        self.key = self._derive_key(settings.ENCRYPTION_KEY.encode())
        self.cipher = Fernet(self.key)
    
    def _derive_key(self, password: bytes) -> bytes:
        """Derive a 32-byte key from password using PBKDF2"""
        # Use a fixed salt for consistency (in production, store salt separately)
        salt = b'enterprise_attendance_salt_2024'
        kdf = PBKDF2HMAC(
            algorithm=hashes.SHA256(),
            length=32,
            salt=salt,
            iterations=100000,
            backend=default_backend()
        )
        key = base64.urlsafe_b64encode(kdf.derive(password))
        return key
    
    def encrypt_vector(self, vector: List[float] | np.ndarray) -> bytes:
        """
        Encrypt a face vector using AES-256.
        
        Args:
            vector: Face vector as list or numpy array
            
        Returns:
            Encrypted vector as bytes (BYTEA for PostgreSQL)
        """
        # Convert to numpy array if needed
        if isinstance(vector, list):
            vector = np.array(vector, dtype=np.float32)
        
        # Serialize vector to bytes
        vector_bytes = vector.tobytes()
        
        # Encrypt
        encrypted = self.cipher.encrypt(vector_bytes)
        
        return encrypted
    
    def decrypt_vector(self, encrypted_vector: bytes) -> List[float]:
        """
        Decrypt a face vector from bytes.
        
        Args:
            encrypted_vector: Encrypted vector bytes from database
            
        Returns:
            Decrypted vector as list of floats
        """
        # Decrypt
        decrypted_bytes = self.cipher.decrypt(encrypted_vector)
        
        # Convert back to numpy array
        vector = np.frombuffer(decrypted_bytes, dtype=np.float32)
        
        return vector.tolist()
    
    def generate_key(self) -> str:
        """Generate a new encryption key (for initial setup)"""
        return Fernet.generate_key().decode()

