# Quick Start Guide - Complete Setup from Zero to Production

This guide will help you set up the entire Enterprise Attendance System step by step.

## Prerequisites Checklist

Before starting, ensure you have:

- [ ] Node.js 20+ installed (`node --version`)
- [ ] Go 1.21+ installed (`go version`)
- [ ] Python 3.11+ installed (`python --version`)
- [ ] A Supabase account (free tier works for development)
- [ ] Git installed
- [ ] A code editor (VS Code recommended)

---

## Step 1: Database Setup (Supabase)

### 1.1 Create Supabase Project

1. Go to https://supabase.com and sign in
2. Click **New Project**
3. Fill in project details:
   - Name: `Glide Attendance System` (or your choice)
   - Database Password: **Save this password!**
   - Region: Choose closest to you
4. Wait 2-3 minutes for project initialization

### 1.2 Run Database Schema

1. In Supabase Dashboard, go to **SQL Editor**
2. Click **New Query**
3. Open `database/schema.sql` from this project
4. Copy **ALL** contents and paste into SQL Editor
5. Click **Run** (Cmd/Ctrl + Enter)
6. Wait for completion (30-60 seconds)

**Verify**: Check **Table Editor** in sidebar - you should see 11 tables.

### 1.3 Enable Extensions

If not already enabled, run in SQL Editor:

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgvector";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
```

### 1.4 Run Auth Integration Migration

1. In SQL Editor, open `database/migrations/001_supabase_auth_integration.sql`
2. Copy and run it
3. This sets up automatic user creation when Supabase auth users sign up

### 1.5 Run RLS Update Migration

1. In SQL Editor, open `database/migrations/002_update_rls_for_supabase.sql`
2. Copy and run it
3. This updates RLS policies to work with Supabase auth

### 1.6 Create Initial Tenant

Run this SQL to create your first tenant:

```sql
INSERT INTO tenants (name, slug, subscription_tier, kiosk_code)
VALUES ('My Organization', 'my-org', 'enterprise', '0000000001')
RETURNING id, name, slug, kiosk_code;
```

**Save the returned `id` (tenant UUID)** - you'll need it later.

### 1.7 Get Database Connection String

1. Go to **Settings** → **Database**
2. Copy the **Connection string** (URI format)
3. Replace `[YOUR-PASSWORD]` with your actual database password
4. Save this for backend configuration

**Example format**:
```
postgresql://postgres:your-password@db.xxxxx.supabase.co:5432/postgres
```

---

## Step 2: Backend Setup (Go API)

### 2.1 Navigate to Backend Directory

```bash
cd backend-golang
```

### 2.2 Create Environment File

```bash
cp .env.example .env
```

### 2.3 Configure Environment Variables

Edit `.env` and fill in:

```bash
# Required
DATABASE_URL=postgresql://postgres:your-password@db.xxxxx.supabase.co:5432/postgres
JWT_SECRET=<generate-with-openssl-rand-base64-32>
ENCRYPTION_KEY=<generate-with-openssl-rand-base64-32>
AI_SERVICE_API_KEY=<generate-with-openssl-rand-base64-32>
KIOSK_HMAC_SECRET=<generate-with-openssl-rand-base64-32>
HRMS_WEBHOOK_SECRET=<generate-with-openssl-rand-base64-32>

# Optional (for now)
AI_SERVICE_URL=http://localhost:8000
CORS_ORIGINS=http://localhost:3000
```

**Generate secrets**:
```bash
openssl rand -base64 32  # Run this 5 times for the secrets above
```

### 2.4 Install Dependencies

```bash
go mod download
```

### 2.5 Test Backend

```bash
go run main.go
```

You should see:
```
Server running on port 8080
```

**Test health endpoint**:
```bash
curl http://localhost:8080/health
```

Should return: `{"status":"ok"}`

---

## Step 3: AI Service Setup (Python FastAPI)

### 3.1 Navigate to AI Directory

```bash
cd ../ai-python
```

### 3.2 Create Python Virtual Environment

```bash
python3.11 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
```

### 3.3 Install Dependencies

```bash
pip install --upgrade pip
pip install -r requirements.txt
```

**Note**: This may take 5-10 minutes as it installs DeepFace and ML libraries.

### 3.4 Create Environment File

```bash
cp .env.example .env
```

### 3.5 Configure Environment Variables

Edit `.env`:

```bash
# Required
DATABASE_URL=postgresql://postgres:your-password@db.xxxxx.supabase.co:5432/postgres
API_KEY=<same-as-AI_SERVICE_API_KEY-from-backend>
ENCRYPTION_KEY=<same-as-ENCRYPTION_KEY-from-backend>

# Optional
HOST=0.0.0.0
PORT=8000
```

### 3.6 Test AI Service

```bash
python main.py
```

Or with uvicorn:
```bash
uvicorn main:app --host 0.0.0.0 --port 8000 --reload
```

You should see:
```
Application startup complete.
Uvicorn running on http://0.0.0.0:8000
```

**Test health endpoint**:
```bash
curl http://localhost:8000/health
```

---

## Step 4: Frontend Setup (Next.js)

### 4.1 Navigate to Frontend Directory

```bash
cd ../frontend-nextjs
```

### 4.2 Install Dependencies

```bash
npm install
```

### 4.3 Create Environment File

```bash
cp .env.example .env.local
```

### 4.4 Configure Environment Variables

Edit `.env.local`:

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_AI_SERVICE_URL=http://localhost:8000
NEXT_PUBLIC_ENABLE_OFFLINE_MODE=true
```

### 4.5 Test Frontend

```bash
npm run dev
```

Open http://localhost:3000 in your browser.

You should see the application homepage.

---

## Step 5: Integration Testing

### 5.1 Start All Services

Open 3 terminal windows:

**Terminal 1 - Backend**:
```bash
cd backend-golang
go run main.go
```

**Terminal 2 - AI Service**:
```bash
cd ai-python
source venv/bin/activate
python main.py
```

**Terminal 3 - Frontend**:
```bash
cd frontend-nextjs
npm run dev
```

### 5.2 Test API Endpoints

**Health Check**:
```bash
curl http://localhost:8080/health
curl http://localhost:8000/health
```

**Login Test** (create a user first via SQL or API):
```bash
curl -X POST http://localhost:8080/api/v1/public/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password"}'
```

---

## Step 6: Create First User

### Option A: Via SQL (Quick)

```sql
-- Replace <TENANT_ID> with your tenant UUID from Step 1.6
INSERT INTO users (
    tenant_id,
    employee_id,
    email,
    first_name,
    last_name,
    date_of_joining,
    role,
    is_active,
    password_hash  -- You'll need to hash this properly
)
VALUES (
    '<TENANT_ID>',
    'ADMIN001',
    'admin@example.com',
    'Admin',
    'User',
    CURRENT_DATE,
    'org_admin',
    true,
    '$2a$10$...'  -- Use bcrypt to hash password
)
RETURNING id, email;
```

### Option B: Via Backend API (Recommended)

Implement user creation endpoint or use a migration script.

---

## Step 7: Production Deployment

### 7.1 Backend Deployment (Render/Koyeb)

1. Push code to GitHub
2. Connect repository to Render/Koyeb
3. Set environment variables (same as `.env`)
4. Deploy

### 7.2 AI Service Deployment (Hugging Face/VPS)

1. Build Docker image or deploy to Hugging Face Spaces
2. Set environment variables
3. Deploy

### 7.3 Frontend Deployment (Vercel)

1. Connect GitHub repository to Vercel
2. Set environment variables (with `NEXT_PUBLIC_` prefix)
3. Deploy

---

## Troubleshooting

### Database Connection Issues

- Verify `DATABASE_URL` is correct
- Check Supabase project is active
- Ensure password doesn't contain special characters (URL encode if needed)

### RLS Policy Errors

- Make sure migrations ran successfully
- Check that session variables are set before queries
- Verify tenant_id exists in database

### AI Service Not Starting

- Check Python version (3.11+)
- Verify all dependencies installed
- Check `ENCRYPTION_KEY` matches backend

### Frontend Build Errors

- Clear `.next` folder: `rm -rf .next`
- Reinstall dependencies: `rm -rf node_modules && npm install`
- Check Node.js version: `node --version` (should be 20+)

---

## Next Steps

1. ✅ Set up CI/CD pipelines (see `.github/workflows/README.md`)
2. ✅ Configure production environment variables
3. ✅ Set up monitoring and logging
4. ✅ Configure backups
5. ✅ Test multi-tenant isolation
6. ✅ Set up SSL certificates
7. ✅ Configure domain names

---

## Getting Help

- Check `database/SETUP_GUIDE.md` for detailed database setup
- Review individual service READMEs:
  - `backend-golang/README.md`
  - `ai-python/README.md`
  - `frontend-nextjs/README.md`
- Check Supabase documentation: https://supabase.com/docs

---

**Congratulations!** Your Enterprise Attendance System is now set up and running locally. 🎉

