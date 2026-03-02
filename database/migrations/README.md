# Database Migrations

This directory contains SQL migration files for setting up and updating the database schema.

## Migration Files

### 001_supabase_auth_integration.sql

Sets up integration between Supabase authentication and the custom users table.

**What it does**:
- Creates functions to extract user metadata from Supabase auth
- Creates triggers to automatically create users in custom `users` table when Supabase auth users sign up
- Handles user updates and deletions
- Provides helper functions for setting user metadata

**When to run**: After running `schema.sql`, before creating any users.

**How to run**:
1. Open Supabase Dashboard → SQL Editor
2. Copy contents of this file
3. Paste and run

### 002_update_rls_for_supabase.sql

Updates Row-Level Security (RLS) policies to work with Supabase authentication.

**What it does**:
- Updates RLS helper functions to use `auth.uid()` when available
- Falls back to session variables for custom JWT tokens
- Recreates all RLS policies with updated functions
- Ensures compatibility with both authentication methods

**When to run**: After running `001_supabase_auth_integration.sql`.

**How to run**:
1. Open Supabase Dashboard → SQL Editor
2. Copy contents of this file
3. Paste and run

## Migration Order

Run migrations in this order:

1. `schema.sql` (main schema)
2. `001_supabase_auth_integration.sql`
3. `002_update_rls_for_supabase.sql`

## Verifying Migrations

After running migrations, verify:

```sql
-- Check functions exist
SELECT proname FROM pg_proc WHERE proname LIKE 'get_%' OR proname LIKE 'handle_%';

-- Check triggers exist
SELECT tgname FROM pg_trigger WHERE tgname LIKE 'on_auth_%';

-- Check RLS is enabled
SELECT tablename, rowsecurity FROM pg_tables WHERE schemaname = 'public';
```

## Troubleshooting

### Error: "permission denied for schema auth"

This is normal - you can't directly modify Supabase's auth schema. The migration uses `SECURITY DEFINER` functions that have elevated privileges.

### Error: "function auth.uid() does not exist"

Make sure you're running this in Supabase, not a local PostgreSQL instance. `auth.uid()` is a Supabase-specific function.

### RLS policies blocking all queries

1. Make sure migrations ran successfully
2. Check that you're setting session variables or using Supabase auth
3. For admin queries, use Supabase service role key

## Creating New Migrations

When creating new migrations:

1. Name them sequentially: `003_description.sql`, `004_description.sql`, etc.
2. Include a header comment explaining what the migration does
3. Make migrations idempotent (use `IF NOT EXISTS`, `DROP IF EXISTS`, etc.)
4. Test migrations on a development database first
5. Document any breaking changes

## Rollback

To rollback a migration:

1. Create a new migration file with rollback SQL
2. Or manually reverse the changes in SQL Editor
3. Always test rollbacks on a development database first


