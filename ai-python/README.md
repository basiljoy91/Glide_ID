# AI Microservice - Enterprise Attendance System

FastAPI-based facial recognition microservice using DeepFace for biometric vectorization and matching.

## Features

- **Face Vectorization**: Convert face images to mathematical vectors using DeepFace
- **1:1 Comparison**: Match a face against a specific user's stored vector
- **1:N Comparison**: Identify a face from all users in a tenant
- **Continuous Learning**: Automatic biometric drift correction (throttled to once per week)
- **Liveness Detection**: Passive and active liveness detection to prevent spoofing
- **AES-256 Encryption**: All face vectors encrypted before storage
- **Health Check**: Lightweight endpoint for frontend silent ping

## Architecture

```
ai-python/
├── main.py                 # FastAPI application entry point
├── config.py              # Configuration and environment variables
├── requirements.txt       # Python dependencies
├── services/
│   ├── face_recognition.py    # DeepFace integration
│   ├── vector_comparison.py   # Similarity calculations
│   ├── continuous_learning.py # Biometric drift correction
│   └── database.py            # PostgreSQL/Supabase connection
└── utils/
    └── encryption.py          # AES-256 encryption utilities
```

## Setup

### 1. Install Dependencies

```bash
pip install -r requirements.txt
```

### 2. Configure Environment

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Key variables to set:
- `DATABASE_URL`: Supabase PostgreSQL connection string
- `API_KEY`: Secure API key for service-to-service authentication
- `ENCRYPTION_KEY`: 32-byte key for AES-256 encryption

### 3. Run the Service

```bash
python main.py
```

Or with uvicorn directly:

```bash
uvicorn main:app --host 0.0.0.0 --port 8000
```

## API Endpoints

### Health Check

```http
GET /health
```

Lightweight endpoint for frontend silent ping. Returns minimal response.

### Vectorize Face

```http
POST /api/v1/vectorize
Content-Type: application/json
X-API-Key: <api-key>

{
  "user_id": "uuid",
  "tenant_id": "uuid",
  "image_base64": "base64-encoded-image",
  "update_existing": false
}
```

Converts a face image to an encrypted vector and stores it in the database.

### Compare Face (1:1)

```http
POST /api/v1/compare
Content-Type: application/json
X-API-Key: <api-key>

{
  "image_base64": "base64-encoded-image",
  "tenant_id": "uuid",
  "user_id": "uuid",
  "threshold": 0.85
}
```

Compares a face image against a specific user's stored vector.

### Compare Face (1:N)

```http
POST /api/v1/compare/multiple
Content-Type: application/json
X-API-Key: <api-key>

{
  "image_base64": "base64-encoded-image",
  "tenant_id": "uuid",
  "threshold": 0.85
}
```

Identifies a face from all users in a tenant. Returns matches sorted by confidence.

### Liveness Detection

```http
POST /api/v1/liveness
Content-Type: application/json
X-API-Key: <api-key>

{
  "image_base64": "base64-encoded-image",
  "liveness_type": "passive"
}
```

Detects if a face is live (real person vs photo/spoof).

## Continuous Learning

The service implements automatic biometric drift correction:

- **Threshold**: Only updates vectors when match confidence ≥ 98%
- **Learning Rate**: Blends 5% new vector with 95% existing vector
- **Throttling**: Maximum once per week per user
- **Automatic**: Triggered during high-confidence matches

## Security

- **API Key Authentication**: All endpoints require `X-API-Key` header
- **AES-256 Encryption**: Face vectors encrypted before database storage
- **Row-Level Security**: Database queries respect tenant isolation
- **No Raw Image Storage**: Only encrypted vectors stored, never raw photos

## Performance

- **Vector Dimension**: 512 (DeepFace VGG-Face default)
- **Similarity Calculation**: Cosine similarity with normalized vectors
- **Database Pooling**: Connection pooling for concurrent requests
- **HNSW Indexing**: Fast similarity search (configured in database)

## Deployment

### Hugging Face Spaces

1. Create a new Space
2. Set environment variables in Settings
3. Deploy with Dockerfile or Python runtime

### VPS/Docker

```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
```

### Environment Variables

Ensure all required environment variables are set:
- `DATABASE_URL`
- `API_KEY`
- `ENCRYPTION_KEY`
- `HOST`, `PORT` (optional, defaults provided)

## Monitoring

Monitor the service via:
- Health check endpoint: `/health`
- FastAPI docs: `/docs` (if DEBUG=true)
- Logs: Check application logs for errors

## Troubleshooting

### DeepFace Model Download

DeepFace automatically downloads models on first use. Ensure:
- Sufficient disk space (~500MB per model)
- Internet connection for initial download
- Models cached in `.DeepFace/` directory

### Database Connection

If connection fails:
- Verify `DATABASE_URL` format
- Check Supabase firewall rules
- Ensure RLS policies allow AI service access

### Vector Comparison Performance

For large tenants (1000+ users):
- Consider HNSW indexing in database
- Implement caching for frequently accessed vectors
- Use batch processing for bulk operations

## License

Proprietary - Enterprise Attendance System

