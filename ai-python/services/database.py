"""
Database Service for PostgreSQL/Supabase
"""

import asyncpg
from typing import Optional, Dict, List, Tuple
from datetime import datetime
import uuid

from config import settings


class DatabaseService:
    """Handle database connections and queries"""
    
    def __init__(self):
        self.pool: Optional[asyncpg.Pool] = None
    
    async def connect(self):
        """Create connection pool"""
        try:
            self.pool = await asyncpg.create_pool(
                settings.DATABASE_URL,
                min_size=2,
                max_size=settings.DB_POOL_SIZE,
                max_inactive_connection_lifetime=300
            )
            print("Database connection pool created")
        except Exception as e:
            print(f"Failed to create database pool: {e}")
            raise
    
    async def disconnect(self):
        """Close connection pool"""
        if self.pool:
            await self.pool.close()
            print("Database connection pool closed")
    
    async def get_face_vector(
        self,
        user_id: str,
        tenant_id: str
    ) -> Optional[bytes]:
        """
        Get encrypted face vector for a user.
        
        Returns:
            Encrypted vector bytes or None
        """
        async with self.pool.acquire() as conn:
            # Set tenant context for RLS
            await conn.execute(
                "SET LOCAL app.current_tenant_id = $1",
                tenant_id
            )
            await conn.execute(
                "SET LOCAL app.is_ai_service = $1",
                'true'
            )
            
            row = await conn.fetchrow(
                """
                SELECT encrypted_vector
                FROM face_vectors
                WHERE user_id = $1 AND tenant_id = $2
                """,
                uuid.UUID(user_id),
                uuid.UUID(tenant_id)
            )
            
            if row:
                return row['encrypted_vector']
            return None
    
    async def store_face_vector(
        self,
        user_id: str,
        tenant_id: str,
        encrypted_vector: bytes,
        vector_dimension: int,
        confidence_score: Optional[float] = None
    ):
        """Store encrypted face vector"""
        async with self.pool.acquire() as conn:
            await conn.execute(
                "SET LOCAL app.current_tenant_id = $1",
                tenant_id
            )
            await conn.execute(
                "SET LOCAL app.is_ai_service = $1",
                'true'
            )
            
            await conn.execute(
                """
                INSERT INTO face_vectors (
                    user_id, tenant_id, encrypted_vector,
                    vector_dimension, confidence_score, created_at, updated_at
                )
                VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
                ON CONFLICT (user_id)
                DO UPDATE SET
                    encrypted_vector = EXCLUDED.encrypted_vector,
                    vector_dimension = EXCLUDED.vector_dimension,
                    confidence_score = EXCLUDED.confidence_score,
                    updated_at = NOW()
                """,
                uuid.UUID(user_id),
                uuid.UUID(tenant_id),
                encrypted_vector,
                vector_dimension,
                confidence_score
            )
    
    async def update_face_vector(
        self,
        user_id: str,
        tenant_id: str,
        encrypted_vector: bytes,
        confidence_score: Optional[float] = None
    ):
        """Update existing face vector (for continuous learning)"""
        async with self.pool.acquire() as conn:
            await conn.execute(
                "SET LOCAL app.current_tenant_id = $1",
                tenant_id
            )
            await conn.execute(
                "SET LOCAL app.is_ai_service = $1",
                'true'
            )
            
            await conn.execute(
                """
                UPDATE face_vectors
                SET
                    encrypted_vector = $1,
                    confidence_score = COALESCE($2, confidence_score),
                    last_learning_update = NOW(),
                    updated_at = NOW()
                WHERE user_id = $3 AND tenant_id = $4
                """,
                encrypted_vector,
                confidence_score,
                uuid.UUID(user_id),
                uuid.UUID(tenant_id)
            )
    
    async def get_all_tenant_vectors(
        self,
        tenant_id: str
    ) -> Dict[str, bytes]:
        """
        Get all encrypted vectors for a tenant (for 1:N matching).
        
        Returns:
            Dictionary mapping user_id -> encrypted_vector
        """
        async with self.pool.acquire() as conn:
            await conn.execute(
                "SET LOCAL app.current_tenant_id = $1",
                tenant_id
            )
            await conn.execute(
                "SET LOCAL app.is_ai_service = $1",
                'true'
            )
            
            rows = await conn.fetch(
                """
                SELECT user_id, encrypted_vector
                FROM face_vectors
                WHERE tenant_id = $1
                """,
                uuid.UUID(tenant_id)
            )
            
            return {
                str(row['user_id']): row['encrypted_vector']
                for row in rows
            }
    
    async def get_user_details(
        self,
        user_id: str,
        tenant_id: str
    ) -> Optional[Dict]:
        """Get user details for match response"""
        async with self.pool.acquire() as conn:
            await conn.execute(
                "SET LOCAL app.current_tenant_id = $1",
                tenant_id
            )
            await conn.execute(
                "SET LOCAL app.is_ai_service = $1",
                'true'
            )
            
            row = await conn.fetchrow(
                """
                SELECT
                    id, employee_id, email, first_name, last_name,
                    department_id, designation
                FROM users
                WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
                """,
                uuid.UUID(user_id),
                uuid.UUID(tenant_id)
            )
            
            if row:
                return {
                    'id': str(row['id']),
                    'employee_id': row['employee_id'],
                    'email': row['email'],
                    'first_name': row['first_name'],
                    'last_name': row['last_name'],
                    'department_id': str(row['department_id']) if row['department_id'] else None,
                    'designation': row['designation']
                }
            return None
    
    async def get_last_learning_update(
        self,
        user_id: str,
        tenant_id: str
    ) -> Optional[datetime]:
        """Get last continuous learning update timestamp"""
        async with self.pool.acquire() as conn:
            await conn.execute(
                "SET LOCAL app.current_tenant_id = $1",
                tenant_id
            )
            await conn.execute(
                "SET LOCAL app.is_ai_service = $1",
                'true'
            )
            
            row = await conn.fetchrow(
                """
                SELECT last_learning_update
                FROM face_vectors
                WHERE user_id = $1 AND tenant_id = $2
                """,
                uuid.UUID(user_id),
                uuid.UUID(tenant_id)
            )
            
            if row and row['last_learning_update']:
                return row['last_learning_update']
            return None

