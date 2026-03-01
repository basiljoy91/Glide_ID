# Enterprise Facial Recognition Attendance & Identity System

A highly secure, multi-tenant SaaS application for enterprise facial recognition attendance and identity management.

## Architecture

This is a monorepo containing all components of the Enterprise Attendance System:

```
enterprise-attendance-system/
├── .github/workflows/    # CI/CD pipelines
├── database/             # PostgreSQL schema
├── frontend-nextjs/      # Next.js frontend (Vercel)
├── backend-golang/       # Go API backend (Render/Koyeb)
└── ai-python/            # Python AI microservice (Hugging Face/VPS)
```

##  Quick Start

### Prerequisites

- Node.js 20+
- Go 1.21+
- Python 3.11+
- PostgreSQL (Supabase)
- Docker (optional)

### Setup

1. **Database Setup**
   ```bash
   cd database
   # Execute schema.sql in Supabase SQL editor
   ```

2. **Backend Setup**
   ```bash
   cd backend-golang
   cp .env.example .env
   # Configure environment variables
   go mod download
   go run main.go
   ```

3. **AI Service Setup**
   ```bash
   cd ai-python
   cp .env.example .env
   # Configure environment variables
   pip install -r requirements.txt
   python main.py
   ```

4. **Frontend Setup**
   ```bash
   cd frontend-nextjs
   cp .env.example .env.local
   # Configure environment variables
   npm install
   npm run dev
   ```

## 📋 Features

### Core Features
- Multi-tenant architecture with Row-Level Security
- Facial recognition using DeepFace
- 1:1 and 1:N face comparison
- Continuous learning (biometric drift correction)
- Offline support with IndexedDB queue
- HMAC signature verification for kiosks
- JWT authentication with SSO support
- HRMS webhook integration
- MQTT IoT door relay control
- Comprehensive audit logging

### Security Features
- ✅ AES-256 encrypted face vectors
- ✅ Row-Level Security (RLS) policies
- ✅ HMAC request signing
- ✅ Asymmetric offline encryption
- ✅ Automated data purging (GDPR/CCPA)
- ✅ Device kill switch for kiosks

### Frontend Features
- ✅ Progressive Web App (PWA)
- ✅ Dark mode support
- ✅ Responsive design
- ✅ Silent ping for backend pre-warming
- ✅ Ambient light detection
- ✅ Camera permission handling
- ✅ Offline queue with automatic sync

## 🔧 Technology Stack

### Frontend
- **Framework**: Next.js 14 (App Router)
- **Styling**: Tailwind CSS
- **State**: Zustand
- **Camera**: WebRTC
- **Offline**: IndexedDB

### Backend
- **Language**: Go 1.21
- **Framework**: Fiber
- **Database**: PostgreSQL (Supabase)
- **MQTT**: Eclipse Paho

### AI Service
- **Language**: Python 3.11
- **Framework**: FastAPI
- **ML**: DeepFace, OpenCV
- **Vector Storage**: pgvector (HNSW)

## 📦 Deployment

### CI/CD Pipelines

The repository includes GitHub Actions workflows with path filtering:

- **Frontend**: Deploys to Vercel on `frontend-nextjs/**` changes
- **Backend**: Deploys to Render/Koyeb on `backend-golang/**` changes
- **AI Service**: Deploys to Hugging Face/VPS on `ai-python/**` changes

See `.github/workflows/README.md` for detailed deployment instructions.

### Manual Deployment

1. **Frontend (Vercel)**
   ```bash
   cd frontend-nextjs
   vercel --prod
   ```

2. **Backend (Render)**
   - Connect GitHub repository
   - Set environment variables
   - Deploy automatically

3. **AI Service (Hugging Face)**
   ```bash
   cd ai-python
   huggingface-cli upload enterprise-attendance-ai .
   ```

## 🔐 Security Configuration

### Required Environment Variables

See individual service READMEs for complete lists:

- **Database**: `DATABASE_URL`
- **Backend**: `JWT_SECRET`, `API_KEY`
- **AI Service**: `ENCRYPTION_KEY`, `API_KEY`
- **Frontend**: `NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_AI_SERVICE_URL`

### Secrets Management

- Never commit secrets to repository
- Use GitHub Secrets for CI/CD
- Use platform-specific secret management (Vercel, Render, etc.)

## 📚 Documentation

- [Database Schema](./database/README.md)
- [Backend API](./backend-golang/README.md)
- [AI Service](./ai-python/README.md)
- [Frontend](./frontend-nextjs/README.md)
- [CI/CD Pipelines](./.github/workflows/README.md)

## 🧪 Testing

```bash
# Frontend
cd frontend-nextjs
npm test

# Backend
cd backend-golang
go test ./...

# AI Service
cd ai-python
pytest
```

## 📝 License

Proprietary - Enterprise Attendance System

## 🤝 Contributing

1. Create feature branch
2. Make changes
3. Ensure CI passes
4. Submit pull request

## 📞 Support

For issues or questions, refer to individual service documentation or contact the development team.

---

**Built with ❤️ for Enterprise Security & Compliance**

