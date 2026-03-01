# Database Schema Documentation

## Overview

This directory contains the PostgreSQL database schema for the Enterprise Facial Recognition Attendance & Identity System, designed to run on Supabase with pgvector extension support.

## Files

- `schema.sql` - Complete database schema with tables, RLS policies, triggers, and functions

## Key Features

### 1. Multi-Tenant Architecture
- Row-Level Security (RLS) ensures complete data isolation between tenants
- All tables include `tenant_id` foreign key references
- RLS policies prevent cross-tenant data access

### 2. Face Vector Storage
- Face vectors stored as AES-256 encrypted BYTEA in `face_vectors` table
- pgvector extension enabled for future HNSW indexing
- Continuous learning support with throttling (max once per week)

### 3. Security Features
- Complete RLS policies on all tables
- Audit logging for all administrative actions
- Automated data purging for GDPR/CCPA compliance (30-day retention)
- HMAC secret management for kiosk authentication

### 4. Core Tables

#### `tenants`
- Organization/tenant information
- Subscription tier management
- 10-digit permanent kiosk code
- SSO configuration

#### `users`
- Employee/user records
- Role-based access control (RBAC)
- Shift timing configuration
- Data privacy consent tracking

#### `departments`
- Department/organizational structure
- Manager assignments

#### `face_vectors`
- Encrypted biometric vectors
- Continuous learning metadata
- One vector per user (updated via learning)

#### `kiosks`
- Kiosk device management
- HMAC secret storage
- IP/geolocation restriction hooks (future)
- MQTT topic configuration for IoT relays

#### `attendance_logs`
- All check-in/check-out records
- Offline time reconciliation (monotonic clock)
- Biometric verification metadata
- PIN fallback tracking

#### `audit_logs`
- Comprehensive audit trail
- Tracks all administrative actions
- User activity logging

#### `hrms_integrations`
- HRMS webhook configuration
- API key/secret management
- Provider-specific settings

#### `payroll_exports`
- Export queue management
- Multiple format support (CSV, Excel, PDF, API)
- Status tracking

#### `offline_queue`
- Encrypted payload storage from IndexedDB
- Sync status tracking
- Retry mechanism

## Deployment Instructions

### 1. Supabase Setup

1. Create a new Supabase project
2. Enable required extensions:
   ```sql
   CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
   CREATE EXTENSION IF NOT EXISTS "pgvector";
   CREATE EXTENSION IF NOT EXISTS "pgcrypto";
   ```

### 2. Run Schema

Execute `schema.sql` in the Supabase SQL editor or via psql:

```bash
psql -h <supabase-host> -U postgres -d postgres -f schema.sql
```

### 3. Configure JWT Claims

In Supabase Dashboard → Authentication → Settings:
- Add custom claims: `tenant_id`, `user_id`, `role`
- Configure RLS to use these claims

### 4. Application Configuration

The application must set session variables before queries:
```sql
SET LOCAL app.current_tenant_id = '<tenant-uuid>';
SET LOCAL app.current_user_id = '<user-uuid>';
SET LOCAL app.is_super_admin = false;
```

### 5. HNSW Index Creation

**Important**: The current schema stores vectors as encrypted BYTEA. For HNSW indexing, you have two options:

**Option A**: Create a decrypted vector column for indexing (less secure, faster):
```sql
ALTER TABLE face_vectors ADD COLUMN decrypted_vector vector(512);
CREATE INDEX ON face_vectors USING hnsw (decrypted_vector vector_cosine_ops)
WITH (m = 16, ef_construction = 64);
```

**Option B**: Handle similarity search entirely in Python AI service (more secure, recommended)

### 6. Automated Purging

To enable automated data purging, install pg_cron extension:
```sql
CREATE EXTENSION IF NOT EXISTS pg_cron;
SELECT cron.schedule(
    'purge-terminated-employees',
    '0 2 * * *',
    'SELECT purge_terminated_employee_data();'
);
```

## Security Considerations

1. **Encryption**: Face vectors must be AES-256 encrypted before storage
2. **RLS**: Never bypass RLS policies. Use service role only for AI microservice
3. **HMAC Secrets**: Generate kiosk HMAC secrets using cryptographically secure random generators
4. **Audit Logs**: Never allow deletion of audit logs (insert-only)
5. **Data Purging**: Ensure automated purging runs daily for compliance

## Performance Optimization

1. **Indexes**: Monitor query performance and add indexes as needed
2. **Partitioning**: Consider partitioning `attendance_logs` by date for large tenants
3. **Connection Pooling**: Use Supabase connection pooling for production
4. **Vector Search**: Optimize HNSW parameters based on data size

## Testing

After deployment, verify:
1. RLS policies prevent cross-tenant access
2. Audit logs capture all administrative actions
3. Automated purging removes old data correctly
4. Continuous learning function updates vectors appropriately

## Support

For issues or questions, refer to the main project documentation or contact the development team.

