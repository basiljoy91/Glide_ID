"""
Enterprise Facial Recognition Attendance & Identity System
AI Microservice - FastAPI Application
"""

from fastapi import FastAPI, HTTPException, UploadFile, File, Depends, Header
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field
from typing import Optional, List
import uvicorn
import os
from datetime import datetime
import base64
import binascii
from io import BytesIO
from PIL import Image, UnidentifiedImageError

from config import settings
from services.face_recognition import FaceRecognitionService
from services.vector_comparison import VectorComparisonService
from services.continuous_learning import ContinuousLearningService
from services.database import DatabaseService
from utils.encryption import EncryptionUtils

app = FastAPI(
    title="Enterprise Attendance AI Service",
    description="Facial Recognition & Biometric Vectorization Microservice",
    version="1.0.0"
)

# CORS Configuration
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.CORS_ORIGINS,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Initialize services
face_recognition = FaceRecognitionService()
vector_comparison = VectorComparisonService()
continuous_learning = ContinuousLearningService()
database = DatabaseService()
encryption = EncryptionUtils()


# ============================================================================
# PYDANTIC MODELS
# ============================================================================

class HealthResponse(BaseModel):
    status: str
    timestamp: datetime
    service: str = "ai-microservice"
    version: str = "1.0.0"


class VectorizeRequest(BaseModel):
    user_id: str = Field(..., description="User UUID")
    tenant_id: str = Field(..., description="Tenant UUID")
    image_base64: Optional[str] = Field(None, description="Base64 encoded image")
    update_existing: bool = Field(False, description="Update existing vector via continuous learning")


class VectorizeResponse(BaseModel):
    success: bool
    user_id: str
    vector_dimension: int
    confidence_score: Optional[float] = None
    message: str


class CompareRequest(BaseModel):
    image_base64: str = Field(..., description="Base64 encoded image to compare")
    tenant_id: str = Field(..., description="Tenant UUID")
    user_id: Optional[str] = Field(None, description="User UUID for 1:1 comparison")
    threshold: float = Field(0.85, description="Similarity threshold (0.0-1.0)")


class CompareResponse(BaseModel):
    match: bool
    confidence: float
    user_id: Optional[str] = None
    user_details: Optional[dict] = None
    message: str


class CompareMultipleResponse(BaseModel):
    matches: List[dict] = Field(..., description="List of matches above threshold")
    total_candidates: int
    processing_time_ms: float


class LivenessRequest(BaseModel):
    image_base64: str = Field(..., description="Base64 encoded image")
    liveness_type: str = Field("passive", description="'active' or 'passive'")
    frames_base64: Optional[List[str]] = Field(None, description="Optional additional frames for active challenge")
    challenge_type: Optional[str] = Field("any", description="active challenge: blink, turn_left, turn_right, move_closer, move_away, any")


class LivenessResponse(BaseModel):
    is_live: bool
    liveness_score: float
    confidence: float
    method: str
    details: Optional[dict] = None


# ============================================================================
# AUTHENTICATION MIDDLEWARE
# ============================================================================

async def verify_api_key(x_api_key: str = Header(...)):
    """Verify API key for service-to-service authentication"""
    if x_api_key != settings.API_KEY:
        raise HTTPException(status_code=401, detail="Invalid API key")
    return x_api_key


# ============================================================================
# HEALTH CHECK ENDPOINT (for Silent Ping)
# ============================================================================

@app.get("/health", response_model=HealthResponse, tags=["Health"])
async def health_check():
    """
    Lightweight health check endpoint for frontend silent ping.
    Returns minimal response to warm up backend on free-tier hosting.
    """
    return HealthResponse(
        status="healthy",
        timestamp=datetime.utcnow()
    )


@app.get("/", tags=["Health"])
async def root():
    """Root endpoint"""
    return {"service": "Enterprise Attendance AI Service", "status": "running"}


# ============================================================================
# FACE VECTORIZATION ENDPOINT
# ============================================================================

@app.post("/api/v1/vectorize", response_model=VectorizeResponse, tags=["Face Recognition"])
async def vectorize_face(
    request: VectorizeRequest,
    api_key: str = Depends(verify_api_key)
):
    """
    Convert a face image to a mathematical vector.
    Supports continuous learning for biometric drift correction.
    
    - **user_id**: UUID of the user
    - **tenant_id**: UUID of the tenant
    - **image_base64**: Base64 encoded image
    - **update_existing**: If true, applies continuous learning to existing vector
    """
    try:
        if not request.image_base64:
            raise HTTPException(status_code=400, detail="image_base64 is required")

        image_b64 = request.image_base64.strip()
        if image_b64.startswith("data:") and "," in image_b64:
            image_b64 = image_b64.split(",", 1)[1]

        try:
            image_data = base64.b64decode(image_b64, validate=True)
        except binascii.Error:
            try:
                image_data = base64.urlsafe_b64decode(image_b64 + "=" * (-len(image_b64) % 4))
            except Exception:
                raise HTTPException(status_code=400, detail="Invalid base64 image payload")

        if not image_data:
            raise HTTPException(status_code=400, detail="Empty image payload")

        if len(image_data) > settings.MAX_IMAGE_SIZE_MB * 1024 * 1024:
            raise HTTPException(
                status_code=400,
                detail=f"Image exceeds max size of {settings.MAX_IMAGE_SIZE_MB}MB"
            )

        try:
            image = Image.open(BytesIO(image_data))
            image.load()
        except UnidentifiedImageError:
            raise HTTPException(status_code=400, detail="Unsupported or invalid image format")
        
        # Convert to RGB if necessary
        if image.mode != 'RGB':
            image = image.convert('RGB')
        
        # Extract face vector
        vector, confidence = await face_recognition.extract_vector(image)
        
        if vector is None:
            raise HTTPException(
                status_code=400,
                detail="No face detected in image. Please ensure face is clearly visible."
            )
        
        # Encrypt vector
        encrypted_vector = encryption.encrypt_vector(vector)
        
        # Check if vector exists for continuous learning
        if request.update_existing:
            existing_vector = await database.get_face_vector(request.user_id, request.tenant_id)
            if existing_vector:
                # Apply continuous learning (blend vectors)
                updated_vector = await continuous_learning.update_vector(
                    existing_vector=existing_vector,
                    new_vector=vector,
                    confidence=confidence
                )
                if updated_vector:
                    encrypted_vector = encryption.encrypt_vector(updated_vector)
                    await database.update_face_vector(
                        user_id=request.user_id,
                        tenant_id=request.tenant_id,
                        encrypted_vector=encrypted_vector,
                        confidence_score=confidence
                    )
                    return VectorizeResponse(
                        success=True,
                        user_id=request.user_id,
                        vector_dimension=len(vector),
                        confidence_score=confidence,
                        message="Vector updated via continuous learning"
                    )
        
        # Store new vector
        await database.store_face_vector(
            user_id=request.user_id,
            tenant_id=request.tenant_id,
            encrypted_vector=encrypted_vector,
            vector_dimension=len(vector),
            confidence_score=confidence
        )
        
        return VectorizeResponse(
            success=True,
            user_id=request.user_id,
            vector_dimension=len(vector),
            confidence_score=confidence,
            message="Face vector stored successfully"
        )
        
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Vectorization failed: {str(e)}")


@app.post("/api/v1/vectorize/file", response_model=VectorizeResponse, tags=["Face Recognition"])
async def vectorize_face_file(
    file: UploadFile = File(...),
    user_id: str = Header(...),
    tenant_id: str = Header(...),
    update_existing: bool = Header(False),
    api_key: str = Depends(verify_api_key)
):
    """
    Vectorize face from uploaded file.
    Alternative endpoint for file uploads instead of base64.
    """
    try:
        from PIL import Image
        from io import BytesIO
        
        # Read image file
        contents = await file.read()
        image = Image.open(BytesIO(contents))
        
        if image.mode != 'RGB':
            image = image.convert('RGB')
        
        # Extract vector
        vector, confidence = await face_recognition.extract_vector(image)
        
        if vector is None:
            raise HTTPException(
                status_code=400,
                detail="No face detected in image"
            )
        
        # Encrypt and store
        encrypted_vector = encryption.encrypt_vector(vector)
        
        if update_existing:
            existing_vector = await database.get_face_vector(user_id, tenant_id)
            if existing_vector:
                updated_vector = await continuous_learning.update_vector(
                    existing_vector=existing_vector,
                    new_vector=vector,
                    confidence=confidence
                )
                if updated_vector:
                    encrypted_vector = encryption.encrypt_vector(updated_vector)
                    await database.update_face_vector(
                        user_id=user_id,
                        tenant_id=tenant_id,
                        encrypted_vector=encrypted_vector,
                        confidence_score=confidence
                    )
                    return VectorizeResponse(
                        success=True,
                        user_id=user_id,
                        vector_dimension=len(vector),
                        confidence_score=confidence,
                        message="Vector updated via continuous learning"
                    )
        
        await database.store_face_vector(
            user_id=user_id,
            tenant_id=tenant_id,
            encrypted_vector=encrypted_vector,
            vector_dimension=len(vector),
            confidence_score=confidence
        )
        
        return VectorizeResponse(
            success=True,
            user_id=user_id,
            vector_dimension=len(vector),
            confidence_score=confidence,
            message="Face vector stored successfully"
        )
        
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Vectorization failed: {str(e)}")


# ============================================================================
# VECTOR COMPARISON ENDPOINTS (1:1 and 1:N)
# ============================================================================

@app.post("/api/v1/compare", response_model=CompareResponse, tags=["Face Recognition"])
async def compare_face(
    request: CompareRequest,
    api_key: str = Depends(verify_api_key)
):
    """
    1:1 Face Comparison
    Compare a face image against a specific user's stored vector.
    
    - **image_base64**: Image to compare
    - **tenant_id**: Tenant UUID
    - **user_id**: User UUID to compare against
    - **threshold**: Similarity threshold (default: 0.85)
    """
    try:
        import base64
        from io import BytesIO
        from PIL import Image
        
        if not request.user_id:
            raise HTTPException(status_code=400, detail="user_id is required for 1:1 comparison")
        
        # Decode image
        image_data = base64.b64decode(request.image_base64)
        image = Image.open(BytesIO(image_data))
        if image.mode != 'RGB':
            image = image.convert('RGB')
        
        # Extract vector from image
        vector, _ = await face_recognition.extract_vector(image)
        if vector is None:
            raise HTTPException(status_code=400, detail="No face detected in image")
        
        # Get stored vector
        encrypted_vector = await database.get_face_vector(request.user_id, request.tenant_id)
        if not encrypted_vector:
            return CompareResponse(
                match=False,
                confidence=0.0,
                message="User vector not found"
            )
        
        # Decrypt stored vector
        stored_vector = encryption.decrypt_vector(encrypted_vector)
        
        # Compare vectors
        similarity = await vector_comparison.cosine_similarity(vector, stored_vector)
        
        match = similarity >= request.threshold
        
        # Get user details if match
        user_details = None
        if match:
            user_details = await database.get_user_details(request.user_id, request.tenant_id)
        
        return CompareResponse(
            match=match,
            confidence=float(similarity),
            user_id=request.user_id if match else None,
            user_details=user_details,
            message="Match found" if match else f"Similarity below threshold ({similarity:.4f} < {request.threshold})"
        )
        
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Comparison failed: {str(e)}")


@app.post("/api/v1/compare/multiple", response_model=CompareMultipleResponse, tags=["Face Recognition"])
async def compare_face_multiple(
    request: CompareRequest,
    api_key: str = Depends(verify_api_key)
):
    """
    1:N Face Comparison (Identification)
    Compare a face image against all users in a tenant.
    Uses HNSW indexing for fast similarity search.
    
    - **image_base64**: Image to compare
    - **tenant_id**: Tenant UUID
    - **threshold**: Similarity threshold (default: 0.85)
    """
    try:
        import base64
        from io import BytesIO
        from PIL import Image
        import time
        
        start_time = time.time()
        
        # Decode image
        image_data = base64.b64decode(request.image_base64)
        image = Image.open(BytesIO(image_data))
        if image.mode != 'RGB':
            image = image.convert('RGB')
        
        # Extract vector from image
        vector, _ = await face_recognition.extract_vector(image)
        if vector is None:
            raise HTTPException(status_code=400, detail="No face detected in image")
        
        # Get all vectors for tenant
        tenant_vectors = await database.get_all_tenant_vectors(request.tenant_id)
        
        if not tenant_vectors:
            return CompareMultipleResponse(
                matches=[],
                total_candidates=0,
                processing_time_ms=0.0
            )
        
        # Compare against all vectors
        matches = []
        for user_id, encrypted_vector in tenant_vectors.items():
            stored_vector = encryption.decrypt_vector(encrypted_vector)
            similarity = await vector_comparison.cosine_similarity(vector, stored_vector)
            
            if similarity >= request.threshold:
                user_details = await database.get_user_details(user_id, request.tenant_id)
                matches.append({
                    "user_id": user_id,
                    "confidence": float(similarity),
                    "user_details": user_details
                })
        
        # Sort by confidence (highest first)
        matches.sort(key=lambda x: x["confidence"], reverse=True)
        
        processing_time = (time.time() - start_time) * 1000  # Convert to milliseconds
        
        return CompareMultipleResponse(
            matches=matches,
            total_candidates=len(tenant_vectors),
            processing_time_ms=processing_time
        )
        
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Multiple comparison failed: {str(e)}")


# ============================================================================
# LIVENESS DETECTION ENDPOINT
# ============================================================================

@app.post("/api/v1/liveness", response_model=LivenessResponse, tags=["Liveness"])
async def detect_liveness(
    request: LivenessRequest,
    api_key: str = Depends(verify_api_key)
):
    """
    Detect if face is live (real person vs photo/spoof).
    
    - **image_base64**: Image to analyze
    - **liveness_type**: 'active' (requires movement) or 'passive' (texture analysis)
    """
    try:
        def decode_b64_image(raw: str) -> Image.Image:
            image_b64 = raw.strip()
            if image_b64.startswith("data:") and "," in image_b64:
                image_b64 = image_b64.split(",", 1)[1]
            image_data = base64.b64decode(image_b64)
            img = Image.open(BytesIO(image_data))
            if img.mode != 'RGB':
                img = img.convert('RGB')
            return img

        image = decode_b64_image(request.image_base64)
        extra_frames: List[Image.Image] = []
        if request.frames_base64:
            for raw in request.frames_base64:
                if not raw:
                    continue
                extra_frames.append(decode_b64_image(raw))

        # Detect liveness with passive + optional active temporal challenge checks
        is_live, score, confidence, details = await face_recognition.detect_liveness(
            image=image,
            liveness_type=request.liveness_type,
            frames=extra_frames,
            challenge_type=request.challenge_type or "any"
        )
        
        return LivenessResponse(
            is_live=is_live,
            liveness_score=float(score),
            confidence=float(confidence),
            method=request.liveness_type,
            details=details
        )
        
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Liveness detection failed: {str(e)}")


# ============================================================================
# CONTINUOUS LEARNING ENDPOINT
# ============================================================================

@app.post("/api/v1/learn", tags=["Continuous Learning"])
async def apply_continuous_learning(
    user_id: str,
    tenant_id: str,
    new_vector: List[float],
    confidence: float,
    api_key: str = Depends(verify_api_key)
):
    """
    Apply continuous learning to update existing face vector.
    Throttled to max once per week per user.
    """
    try:
        # Get existing vector
        encrypted_vector = await database.get_face_vector(user_id, tenant_id)
        if not encrypted_vector:
            raise HTTPException(status_code=404, detail="User vector not found")
        
        stored_vector = encryption.decrypt_vector(encrypted_vector)
        
        # Check if update is allowed (throttling)
        can_update = await continuous_learning.can_update(user_id, tenant_id)
        if not can_update:
            return JSONResponse(
                status_code=200,
                content={
                    "success": False,
                    "message": "Continuous learning update throttled (max once per week)"
                }
            )
        
        # Update vector
        updated_vector = await continuous_learning.update_vector(
            existing_vector=stored_vector,
            new_vector=new_vector,
            confidence=confidence
        )
        
        if updated_vector:
            encrypted_updated = encryption.encrypt_vector(updated_vector)
            await database.update_face_vector(
                user_id=user_id,
                tenant_id=tenant_id,
                encrypted_vector=encrypted_updated,
                confidence_score=confidence
            )
            return JSONResponse(
                status_code=200,
                content={
                    "success": True,
                    "message": "Vector updated via continuous learning"
                }
            )
        
        return JSONResponse(
            status_code=200,
            content={
                "success": False,
                "message": "Update not applied (confidence threshold not met)"
            }
        )
        
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Continuous learning failed: {str(e)}")


# ============================================================================
# ERROR HANDLERS
# ============================================================================

@app.exception_handler(HTTPException)
async def http_exception_handler(request, exc):
    return JSONResponse(
        status_code=exc.status_code,
        content={"error": exc.detail}
    )


@app.exception_handler(Exception)
async def general_exception_handler(request, exc):
    return JSONResponse(
        status_code=500,
        content={"error": "Internal server error", "detail": str(exc)}
    )


# ============================================================================
# STARTUP/SHUTDOWN EVENTS
# ============================================================================

@app.on_event("startup")
async def startup_event():
    """Initialize services on startup"""
    await database.connect()
    print("AI Microservice started successfully")


@app.on_event("shutdown")
async def shutdown_event():
    """Cleanup on shutdown"""
    await database.disconnect()
    print("AI Microservice shut down")


# ============================================================================
# MAIN ENTRY POINT
# ============================================================================

if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=settings.DEBUG,
        log_level="info"
    )
