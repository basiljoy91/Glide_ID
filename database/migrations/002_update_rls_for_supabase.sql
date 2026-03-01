-- ============================================================================
-- Migration: Update RLS Policies for Supabase Auth Compatibility
-- Description: Updates RLS functions to work with both Supabase auth and custom JWTs
-- Date: 2024
-- ============================================================================
-- This migration updates the RLS helper functions to:
-- 1. Use Supabase auth.uid() when available (Supabase auth users)
-- 2. Fall back to session variables for custom JWT tokens
-- 3. Ensure compatibility with both authentication methods
-- ============================================================================

-- ============================================================================
-- UPDATE: Enhanced get_user_tenant_id() function
-- ============================================================================

CREATE OR REPLACE FUNCTION get_user_tenant_id()
RETURNS UUID AS $$
DECLARE
    v_tenant_id UUID;
BEGIN
    -- First, try to get tenant_id from Supabase auth user metadata
    BEGIN
        SELECT (raw_user_meta_data->>'tenant_id')::UUID INTO v_tenant_id
        FROM auth.users
        WHERE id = auth.uid();
        
        IF v_tenant_id IS NOT NULL THEN
            RETURN v_tenant_id;
        END IF;
    EXCEPTION
        WHEN OTHERS THEN
            -- Supabase auth not available or user not authenticated
            NULL;
    END;
    
    -- Fall back to session variable (for custom JWT tokens)
    BEGIN
        RETURN current_setting('app.current_tenant_id', true)::UUID;
    EXCEPTION
        WHEN OTHERS THEN
            RETURN NULL;
    END;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- UPDATE: Enhanced get_user_id() function
-- ============================================================================

CREATE OR REPLACE FUNCTION get_user_id()
RETURNS UUID AS $$
DECLARE
    v_user_id UUID;
BEGIN
    -- First, try to get user_id from Supabase auth
    BEGIN
        SELECT auth.uid() INTO v_user_id;
        
        IF v_user_id IS NOT NULL THEN
            RETURN v_user_id;
        END IF;
    EXCEPTION
        WHEN OTHERS THEN
            -- Supabase auth not available
            NULL;
    END;
    
    -- Fall back to session variable (for custom JWT tokens)
    BEGIN
        RETURN current_setting('app.current_user_id', true)::UUID;
    EXCEPTION
        WHEN OTHERS THEN
            RETURN NULL;
    END;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- UPDATE: Enhanced is_super_admin() function
-- ============================================================================

CREATE OR REPLACE FUNCTION is_super_admin()
RETURNS BOOLEAN AS $$
DECLARE
    v_role TEXT;
    v_is_super_admin BOOLEAN;
BEGIN
    -- First, try to get role from Supabase auth user metadata
    BEGIN
        SELECT raw_user_meta_data->>'role' INTO v_role
        FROM auth.users
        WHERE id = auth.uid();
        
        IF v_role = 'super_admin' THEN
            RETURN true;
        END IF;
    EXCEPTION
        WHEN OTHERS THEN
            -- Supabase auth not available
            NULL;
    END;
    
    -- Fall back to session variable (for custom JWT tokens)
    BEGIN
        RETURN current_setting('app.is_super_admin', true)::BOOLEAN = true;
    EXCEPTION
        WHEN OTHERS THEN
            RETURN false;
    END;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- UPDATE: Enhanced RLS policies to use new functions
-- ============================================================================

-- Drop existing policies (they will be recreated with updated functions)
-- Note: We'll update key policies, but you may need to review all policies

-- ============================================================================
-- TENANTS RLS POLICIES (Updated)
-- ============================================================================

-- Drop and recreate policies with updated functions
DROP POLICY IF EXISTS "super_admin_read_tenants" ON tenants;
DROP POLICY IF EXISTS "org_admin_read_own_tenant" ON tenants;
DROP POLICY IF EXISTS "super_admin_update_tenants" ON tenants;
DROP POLICY IF EXISTS "org_admin_update_own_tenant" ON tenants;

-- Super admins can read all tenants
CREATE POLICY "super_admin_read_tenants" ON tenants
    FOR SELECT
    USING (is_super_admin());

-- Org admins can read their own tenant
CREATE POLICY "org_admin_read_own_tenant" ON tenants
    FOR SELECT
    USING (id = get_user_tenant_id());

-- Super admins can update all tenants
CREATE POLICY "super_admin_update_tenants" ON tenants
    FOR UPDATE
    USING (is_super_admin());

-- Org admins can update their own tenant
CREATE POLICY "org_admin_update_own_tenant" ON tenants
    FOR UPDATE
    USING (id = get_user_tenant_id());

-- ============================================================================
-- USERS RLS POLICIES (Updated)
-- ============================================================================

-- Drop and recreate key policies
DROP POLICY IF EXISTS "super_admin_read_users" ON users;
DROP POLICY IF EXISTS "tenant_users_read" ON users;
DROP POLICY IF EXISTS "users_read_own" ON users;
DROP POLICY IF EXISTS "hr_insert_users" ON users;
DROP POLICY IF EXISTS "hr_update_users" ON users;
DROP POLICY IF EXISTS "hr_delete_users" ON users;

-- Super admins can read all users
CREATE POLICY "super_admin_read_users" ON users
    FOR SELECT
    USING (is_super_admin());

-- Users can read users in their tenant
CREATE POLICY "tenant_users_read" ON users
    FOR SELECT
    USING (tenant_id = get_user_tenant_id() AND deleted_at IS NULL);

-- Users can read their own record
CREATE POLICY "users_read_own" ON users
    FOR SELECT
    USING (id = get_user_id());

-- HR/Admins can insert users in their tenant
CREATE POLICY "hr_insert_users" ON users
    FOR INSERT
    WITH CHECK (tenant_id = get_user_tenant_id());

-- HR/Admins can update users in their tenant
CREATE POLICY "hr_update_users" ON users
    FOR UPDATE
    USING (tenant_id = get_user_tenant_id());

-- HR/Admins can soft-delete users in their tenant
CREATE POLICY "hr_delete_users" ON users
    FOR UPDATE
    USING (tenant_id = get_user_tenant_id())
    WITH CHECK (deleted_at IS NOT NULL);

-- ============================================================================
-- DEPARTMENTS RLS POLICIES (Updated)
-- ============================================================================

DROP POLICY IF EXISTS "tenant_departments_read" ON departments;
DROP POLICY IF EXISTS "hr_insert_departments" ON departments;
DROP POLICY IF EXISTS "hr_update_departments" ON departments;
DROP POLICY IF EXISTS "hr_delete_departments" ON departments;

CREATE POLICY "tenant_departments_read" ON departments
    FOR SELECT
    USING (tenant_id = get_user_tenant_id() AND deleted_at IS NULL);

CREATE POLICY "hr_insert_departments" ON departments
    FOR INSERT
    WITH CHECK (tenant_id = get_user_tenant_id());

CREATE POLICY "hr_update_departments" ON departments
    FOR UPDATE
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "hr_delete_departments" ON departments
    FOR UPDATE
    USING (tenant_id = get_user_tenant_id())
    WITH CHECK (deleted_at IS NOT NULL);

-- ============================================================================
-- EMERGENCY CONTACTS RLS POLICIES (Updated)
-- ============================================================================

DROP POLICY IF EXISTS "users_read_own_contacts" ON emergency_contacts;
DROP POLICY IF EXISTS "hr_insert_contacts" ON emergency_contacts;
DROP POLICY IF EXISTS "hr_update_contacts" ON emergency_contacts;

CREATE POLICY "users_read_own_contacts" ON emergency_contacts
    FOR SELECT
    USING (
        user_id = get_user_id()
        OR user_id IN (
            SELECT id FROM users WHERE tenant_id = get_user_tenant_id()
        )
    );

CREATE POLICY "hr_insert_contacts" ON emergency_contacts
    FOR INSERT
    WITH CHECK (
        user_id IN (SELECT id FROM users WHERE tenant_id = get_user_tenant_id())
    );

CREATE POLICY "hr_update_contacts" ON emergency_contacts
    FOR UPDATE
    USING (
        user_id IN (SELECT id FROM users WHERE tenant_id = get_user_tenant_id())
    );

-- ============================================================================
-- FACE VECTORS RLS POLICIES (Updated)
-- ============================================================================

DROP POLICY IF EXISTS "tenant_face_vectors_read" ON face_vectors;
DROP POLICY IF EXISTS "ai_service_manage_vectors" ON face_vectors;

CREATE POLICY "tenant_face_vectors_read" ON face_vectors
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "ai_service_manage_vectors" ON face_vectors
    FOR ALL
    USING (current_setting('app.is_ai_service', true)::BOOLEAN = true);

-- ============================================================================
-- KIOSKS RLS POLICIES (Updated)
-- ============================================================================

DROP POLICY IF EXISTS "tenant_kiosks_read" ON kiosks;
DROP POLICY IF EXISTS "admin_insert_kiosks" ON kiosks;
DROP POLICY IF EXISTS "admin_update_kiosks" ON kiosks;

CREATE POLICY "tenant_kiosks_read" ON kiosks
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "admin_insert_kiosks" ON kiosks
    FOR INSERT
    WITH CHECK (tenant_id = get_user_tenant_id());

CREATE POLICY "admin_update_kiosks" ON kiosks
    FOR UPDATE
    USING (tenant_id = get_user_tenant_id());

-- ============================================================================
-- ATTENDANCE LOGS RLS POLICIES (Updated)
-- ============================================================================

DROP POLICY IF EXISTS "tenant_attendance_read" ON attendance_logs;
DROP POLICY IF EXISTS "users_read_own_attendance" ON attendance_logs;
DROP POLICY IF EXISTS "kiosk_insert_attendance" ON attendance_logs;
DROP POLICY IF EXISTS "hr_update_attendance" ON attendance_logs;

CREATE POLICY "tenant_attendance_read" ON attendance_logs
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "users_read_own_attendance" ON attendance_logs
    FOR SELECT
    USING (user_id = get_user_id());

CREATE POLICY "kiosk_insert_attendance" ON attendance_logs
    FOR INSERT
    WITH CHECK (
        tenant_id = get_user_tenant_id()
        AND kiosk_id IN (SELECT id FROM kiosks WHERE tenant_id = get_user_tenant_id())
    );

CREATE POLICY "hr_update_attendance" ON attendance_logs
    FOR UPDATE
    USING (tenant_id = get_user_tenant_id());

-- ============================================================================
-- AUDIT LOGS RLS POLICIES (Updated)
-- ============================================================================

DROP POLICY IF EXISTS "tenant_audit_read" ON audit_logs;
DROP POLICY IF EXISTS "system_insert_audit" ON audit_logs;

CREATE POLICY "tenant_audit_read" ON audit_logs
    FOR SELECT
    USING (tenant_id = get_user_tenant_id() OR tenant_id IS NULL);

CREATE POLICY "system_insert_audit" ON audit_logs
    FOR INSERT
    WITH CHECK (true); -- Application layer enforces tenant_id

-- ============================================================================
-- HRMS INTEGRATIONS RLS POLICIES (Updated)
-- ============================================================================

DROP POLICY IF EXISTS "tenant_hrms_read" ON hrms_integrations;
DROP POLICY IF EXISTS "admin_manage_hrms" ON hrms_integrations;

CREATE POLICY "tenant_hrms_read" ON hrms_integrations
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "admin_manage_hrms" ON hrms_integrations
    FOR ALL
    USING (tenant_id = get_user_tenant_id());

-- ============================================================================
-- PAYROLL EXPORTS RLS POLICIES (Updated)
-- ============================================================================

DROP POLICY IF EXISTS "tenant_payroll_read" ON payroll_exports;
DROP POLICY IF EXISTS "hr_create_payroll_export" ON payroll_exports;
DROP POLICY IF EXISTS "system_update_payroll_export" ON payroll_exports;

CREATE POLICY "tenant_payroll_read" ON payroll_exports
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "hr_create_payroll_export" ON payroll_exports
    FOR INSERT
    WITH CHECK (tenant_id = get_user_tenant_id());

CREATE POLICY "system_update_payroll_export" ON payroll_exports
    FOR UPDATE
    USING (tenant_id = get_user_tenant_id());

-- ============================================================================
-- OFFLINE QUEUE RLS POLICIES (Updated)
-- ============================================================================

DROP POLICY IF EXISTS "tenant_offline_queue_read" ON offline_queue;
DROP POLICY IF EXISTS "kiosk_insert_offline_queue" ON offline_queue;
DROP POLICY IF EXISTS "system_update_offline_queue" ON offline_queue;

CREATE POLICY "tenant_offline_queue_read" ON offline_queue
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "kiosk_insert_offline_queue" ON offline_queue
    FOR INSERT
    WITH CHECK (
        tenant_id = get_user_tenant_id()
        AND kiosk_id IN (SELECT id FROM kiosks WHERE tenant_id = get_user_tenant_id())
    );

CREATE POLICY "system_update_offline_queue" ON offline_queue
    FOR UPDATE
    USING (tenant_id = get_user_tenant_id());

-- ============================================================================
-- NOTES
-- ============================================================================
-- 1. These updated functions work with both Supabase auth and custom JWTs
-- 2. For Supabase auth users, tenant_id should be set in raw_user_meta_data
-- 3. For custom JWT tokens, use session variables (app.current_tenant_id, etc.)
-- 4. Test both authentication methods to ensure RLS works correctly
-- 5. The is_super_admin() function checks both auth metadata and session variables
-- ============================================================================

