# Complete Database Setup Guide for Supabase

This guide will walk you through setting up your database from scratch to production-ready state.

## Prerequisites

- A Supabase account (sign up at https://supabase.com)
- Basic knowledge of SQL
- Access to your Supabase project dashboard

---

## Step 1: Create Supabase Project

1. Go to https://supabase.com and sign in
2. Click "New Project"
3. Fill in:
   - **Name**: Your project name (e.g., "Glide Attendance System")
   - **Database Password**: Choose a strong password (save this!)
   - **Region**: Choose closest to your users
   - **Pricing Plan**: Start with Free tier for development
4. Click "Create new project"
5. Wait 2-3 minutes for project to initialize

---

## Step 2: Enable Required Extensions

1. In Supabase Dashboard, go to **SQL Editor** (left sidebar)
2. Click **New Query**
3. Run this SQL to enable required extensions:

```sql
-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgvector";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
```

4. Click **Run** (or press Cmd/Ctrl + Enter)
5. You should see "Success. No rows returned"

**Note**: If `pgvector` extension is not available, you may need to request it from Supabase support or use a different Supabase plan.

---

## Step 3: Run Main Schema

1. In SQL Editor, click **New Query**
2. Open the file `database/schema.sql` from this project
3. Copy **ALL** contents of `schema.sql`
4. Paste into the SQL Editor
5. Click **Run**
6. Wait for execution to complete (may take 30-60 seconds)
7. You should see "Success. No rows returned" at the end

**Verify**: Check the left sidebar → **Table Editor**. You should see these tables:
- `tenants`
- `users`
- `departments`
- `emergency_contacts`
- `face_vectors`
- `kiosks`
- `attendance_logs`
- `audit_logs`
- `hrms_integrations`
- `payroll_exports`
- `offline_queue`

---

## Step 4: Set Up Supabase Auth Integration (Optional but Recommended)

Since Supabase doesn't have a UI for custom claims, we'll create database functions that work with Supabase's built-in authentication.

### 4.1: Run Auth Integration Migration

1. In SQL Editor, click **New Query**
2. Open and run `database/migrations/001_supabase_auth_integration.sql` (we'll create this next)
3. This creates functions that link Supabase auth users to your custom users table

### 4.2: Create Database Trigger for User Creation

The migration file will create a trigger that automatically:
- Links Supabase auth users to your `users` table
- Sets up user metadata when users sign up
- Handles user updates

---

## Step 5: Configure RLS Policies for Supabase Auth

The schema already includes RLS policies, but we need to update them to work with Supabase's `auth.uid()` function.

1. Run `database/migrations/002_update_rls_for_supabase.sql` (we'll create this)

This migration:
- Updates RLS functions to use `auth.uid()` when available
- Falls back to session variables for custom JWT tokens
- Ensures compatibility with both authentication methods

---

## Step 6: Create Initial Tenant and Super Admin

You need at least one tenant to start using the system.

### Option A: Via SQL (Quick Start)

```sql
-- Create a test tenant
INSERT INTO tenants (name, slug, subscription_tier, kiosk_code)
VALUES ('Demo Organization', 'demo-org', 'enterprise', '0000000001')
RETURNING id, name, slug, kiosk_code;

-- Note the tenant_id from the output above, then create a super admin user
-- Replace <TENANT_ID> with the UUID from above
INSERT INTO users (
    tenant_id,
    employee_id,
    email,
    first_name,
    last_name,
    date_of_joining,
    role,
    is_active
)
VALUES (
    '<TENANT_ID>',  -- Replace with actual tenant UUID
    'ADMIN001',
    'admin@example.com',
    'Super',
    'Admin',
    CURRENT_DATE,
    'super_admin',
    true
)
RETURNING id, email, role;
```

### Option B: Via Application (Recommended for Production)

Use your backend API to create tenants and users through proper application logic.

---

## Step 7: Get Database Connection Details

1. In Supabase Dashboard, go to **Settings** → **Database**
2. Find **Connection string** section
3. Copy the **URI** connection string (looks like: `postgresql://postgres:[YOUR-PASSWORD]@db.xxx.supabase.co:5432/postgres`)
4. Save this for your backend configuration

**Important**: 
- Replace `[YOUR-PASSWORD]` with your actual database password
- Use **Connection Pooling** URL for production (found in Settings → Database → Connection Pooling)
- Use **Direct Connection** URL for migrations and admin tasks

---

## Step 8: Test Database Connection

### Using psql (Command Line)

```bash
# Install psql if needed (macOS)
brew install postgresql

# Connect to Supabase
psql "postgresql://postgres:[YOUR-PASSWORD]@db.xxx.supabase.co:5432/postgres"

# Test query
SELECT COUNT(*) FROM tenants;

# Exit
\q
```

### Using Supabase Dashboard

1. Go to **SQL Editor**
2. Run: `psql`
3. Should return a number (0 or more)

---

## Step 9: Set Up Automated Tasks (Optional)

### Enable pg_cron for Automated Purging

```sql
-- Enable pg_cron extension (may require Supabase support)
CREATE EXTENSION IF NOT EXISTS pg_cron;

-- Schedule daily data purging at 2 AM
SELECT cron.schedule(
    'purge-terminated-employees',
    '0 2 * * *',
    'SELECT purge_terminated_employee_data();'
);
```

**Note**: pg_cron may not be available on free tier. Contact Supabase support if needed.

---

## Step 10: Verify RLS Policies

Test that Row-Level Security is working:

```sql
-- Test 1: Try to read tenants without setting tenant context
SELECT * FROM tenants;
-- Should return empty or error (depending on RLS policy)

-- Test 2: Set tenant context and read
SET LOCAL app.current_tenant_id = '<YOUR-TENANT-ID>';
SELECT * FROM tenants WHERE id = '<YOUR-TENANT-ID>';
-- Should return your tenant

-- Test 3: Try to read another tenant's data
SET LOCAL app.current_tenant_id = '00000000-0000-0000-0000-000000000000';
SELECT * FROM tenants WHERE id = '<YOUR-TENANT-ID>';
-- Should return empty (RLS blocking cross-tenant access)
```

---

## Step 11: Create Storage Buckets (For File Uploads)

1. Go to **Storage** in Supabase Dashboard
2. Create buckets:
   - `payroll-exports` (public: false)
   - `face-images` (public: false) - if storing face images
   - `audit-reports` (public: false)

For each bucket:
- Click **New bucket**
- Enter bucket name
- Set **Public bucket** to OFF (private)
- Click **Create bucket**

---

## Step 12: Set Up Database Backups

1. Go to **Settings** → **Database**
2. Scroll to **Backups** section
3. Enable **Point-in-time Recovery** (if available on your plan)
4. Set backup retention period

**Note**: Free tier may have limited backup options. Consider upgrading for production.

---

## Troubleshooting

### Issue: "Extension pgvector does not exist"

**Solution**: 
- Contact Supabase support to enable pgvector
- Or use a paid plan that includes pgvector
- For development, you can temporarily comment out pgvector-related code

### Issue: "Permission denied for schema auth"

**Solution**: 
- This is normal - you can't directly modify Supabase's auth schema
- Use the migration files we provide instead
- They use Supabase's public API functions

### Issue: RLS policies blocking all queries

**Solution**:
- Make sure you're setting session variables before queries
- Check that your backend is calling `SetTenantContext()` and `SetUserContext()`
- For admin queries, use Supabase service role key (bypasses RLS)

### Issue: Can't create users in Supabase Auth

**Solution**:
- Go to **Authentication** → **Users** in dashboard
- Click **Add user** → **Create new user**
- Or use Supabase Auth API from your backend

---

## Next Steps

After completing database setup:

1. ✅ Configure backend environment variables (see `backend-golang/README.md`)
2. ✅ Configure AI service environment variables (see `ai-python/README.md`)
3. ✅ Configure frontend environment variables (see `frontend-nextjs/README.md`)
4. ✅ Test API endpoints
5. ✅ Set up CI/CD pipelines (see `.github/workflows/README.md`)

---

## Production Checklist

Before going to production:

- [ ] Enable database backups
- [ ] Set up connection pooling
- [ ] Configure proper RLS policies
- [ ] Test multi-tenant isolation
- [ ] Set up monitoring and alerts
- [ ] Enable automated purging (pg_cron)
- [ ] Review and test all RLS policies
- [ ] Set up database migrations workflow
- [ ] Document all custom functions
- [ ] Test disaster recovery procedures

---

## Support

If you encounter issues:
1. Check Supabase documentation: https://supabase.com/docs
2. Review error messages in Supabase Dashboard → Logs
3. Check project GitHub issues
4. Contact Supabase support for platform-specific issues

