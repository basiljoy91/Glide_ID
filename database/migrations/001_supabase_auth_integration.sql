-- ============================================================================
-- Migration: Supabase Auth Integration
-- Description: Links Supabase authentication with custom users table
-- Date: 2024
-- ============================================================================
-- This migration creates functions and triggers to:
-- 1. Automatically create users in custom users table when Supabase auth users sign up
-- 2. Sync user metadata between Supabase auth and custom users table
-- 3. Handle user updates and deletions
-- ============================================================================

-- ============================================================================
-- FUNCTION: Get user metadata from Supabase auth
-- ============================================================================

-- Function to get tenant_id from Supabase auth user metadata
CREATE OR REPLACE FUNCTION get_auth_user_tenant_id()
RETURNS UUID AS $$
BEGIN
    -- Try to get tenant_id from auth.users.raw_user_meta_data
    -- This requires the application to set it during user creation
    RETURN (
        SELECT (raw_user_meta_data->>'tenant_id')::UUID
        FROM auth.users
        WHERE id = auth.uid()
    );
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Function to get user_id from Supabase auth
CREATE OR REPLACE FUNCTION get_auth_user_id()
RETURNS UUID AS $$
BEGIN
    RETURN auth.uid();
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- Function to check if current user is super admin
CREATE OR REPLACE FUNCTION get_auth_user_role()
RETURNS TEXT AS $$
BEGIN
    -- Try to get role from auth.users.raw_user_meta_data
    RETURN (
        SELECT raw_user_meta_data->>'role'
        FROM auth.users
        WHERE id = auth.uid()
    );
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE SECURITY DEFINER;

-- ============================================================================
-- FUNCTION: Create user in custom users table from Supabase auth
-- ============================================================================

CREATE OR REPLACE FUNCTION handle_new_user()
RETURNS TRIGGER AS $$
DECLARE
    v_tenant_id UUID;
    v_user_id UUID;
    v_email TEXT;
    v_role TEXT;
BEGIN
    -- Extract metadata from Supabase auth user
    v_tenant_id := (NEW.raw_user_meta_data->>'tenant_id')::UUID;
    v_email := NEW.email;
    v_role := COALESCE(NEW.raw_user_meta_data->>'role', 'employee');
    
    -- If tenant_id is not provided, we can't create the user
    -- This should be set by the application during signup
    IF v_tenant_id IS NULL THEN
        RAISE WARNING 'Tenant ID not provided for user: %', v_email;
        RETURN NEW;
    END IF;
    
    -- Check if user already exists in custom users table
    SELECT id INTO v_user_id
    FROM users
    WHERE email = v_email AND tenant_id = v_tenant_id;
    
    -- If user doesn't exist, create it
    IF v_user_id IS NULL THEN
        INSERT INTO users (
            id,  -- Use same UUID as auth.users.id for consistency
            tenant_id,
            employee_id,
            email,
            first_name,
            last_name,
            date_of_joining,
            role,
            is_active,
            password_hash  -- Not used if using Supabase auth, but kept for compatibility
        )
        VALUES (
            NEW.id,
            v_tenant_id,
            COALESCE(
                NEW.raw_user_meta_data->>'employee_id',
                'EMP' || LPAD(EXTRACT(EPOCH FROM NOW())::TEXT, 10, '0')
            ),
            v_email,
            COALESCE(NEW.raw_user_meta_data->>'first_name', ''),
            COALESCE(NEW.raw_user_meta_data->>'last_name', ''),
            COALESCE(
                (NEW.raw_user_meta_data->>'date_of_joining')::DATE,
                CURRENT_DATE
            ),
            v_role::user_role,
            COALESCE((NEW.raw_user_meta_data->>'is_active')::BOOLEAN, true),
            NULL  -- Password handled by Supabase auth
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- ============================================================================
-- TRIGGER: Auto-create user when Supabase auth user is created
-- ============================================================================

-- Drop trigger if exists
DROP TRIGGER IF EXISTS on_auth_user_created ON auth.users;

-- Create trigger
CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION handle_new_user();

-- ============================================================================
-- FUNCTION: Update user metadata when Supabase auth user is updated
-- ============================================================================

CREATE OR REPLACE FUNCTION handle_user_update()
RETURNS TRIGGER AS $$
BEGIN
    -- Update email if changed
    IF OLD.email IS DISTINCT FROM NEW.email THEN
        UPDATE users
        SET email = NEW.email,
            updated_at = NOW()
        WHERE id = NEW.id;
    END IF;
    
    -- Update metadata if changed
    IF OLD.raw_user_meta_data IS DISTINCT FROM NEW.raw_user_meta_data THEN
        UPDATE users
        SET 
            first_name = COALESCE(
                (NEW.raw_user_meta_data->>'first_name')::VARCHAR(100),
                first_name
            ),
            last_name = COALESCE(
                (NEW.raw_user_meta_data->>'last_name')::VARCHAR(100),
                last_name
            ),
            role = COALESCE(
                (NEW.raw_user_meta_data->>'role')::user_role,
                role
            ),
            is_active = COALESCE(
                (NEW.raw_user_meta_data->>'is_active')::BOOLEAN,
                is_active
            ),
            updated_at = NOW()
        WHERE id = NEW.id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- ============================================================================
-- TRIGGER: Update user when Supabase auth user is updated
-- ============================================================================

DROP TRIGGER IF EXISTS on_auth_user_updated ON auth.users;

CREATE TRIGGER on_auth_user_updated
    AFTER UPDATE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION handle_user_update();

-- ============================================================================
-- FUNCTION: Soft delete user when Supabase auth user is deleted
-- ============================================================================

CREATE OR REPLACE FUNCTION handle_user_delete()
RETURNS TRIGGER AS $$
BEGIN
    -- Soft delete user in custom users table
    UPDATE users
    SET deleted_at = NOW(),
        is_active = false,
        updated_at = NOW()
    WHERE id = OLD.id;
    
    RETURN OLD;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- ============================================================================
-- TRIGGER: Soft delete user when Supabase auth user is deleted
-- ============================================================================

DROP TRIGGER IF EXISTS on_auth_user_deleted ON auth.users;

CREATE TRIGGER on_auth_user_deleted
    AFTER DELETE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION handle_user_delete();

-- ============================================================================
-- HELPER FUNCTION: Set user metadata in Supabase auth
-- ============================================================================
-- This function can be called from your application to set user metadata
-- Example: SELECT set_auth_user_metadata('tenant_id', '<uuid>'::TEXT);

CREATE OR REPLACE FUNCTION set_auth_user_metadata(
    p_key TEXT,
    p_value TEXT,
    p_user_id UUID DEFAULT auth.uid()
)
RETURNS void AS $$
BEGIN
    -- Update raw_user_meta_data in auth.users
    UPDATE auth.users
    SET raw_user_meta_data = jsonb_set(
        COALESCE(raw_user_meta_data, '{}'::jsonb),
        ARRAY[p_key],
        to_jsonb(p_value)
    )
    WHERE id = p_user_id;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- ============================================================================
-- NOTES
-- ============================================================================
-- 1. These functions use SECURITY DEFINER to access auth.users table
-- 2. The application should set tenant_id in raw_user_meta_data during signup
-- 3. For custom JWT tokens (not using Supabase auth), use the session variable approach
-- 4. Test these functions with a test user before production deployment
-- ============================================================================

