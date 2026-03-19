-- ============================================================================
-- Enterprise Facial Recognition Attendance & Identity System
-- Database Schema for PostgreSQL (Supabase)
-- ============================================================================
-- This schema includes:
-- - pgvector extension for face vector storage
-- - HNSW indexing for fast similarity search
-- - Row-Level Security (RLS) for multi-tenant isolation
-- - Complete audit logging
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgvector";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- ENUMS
-- ============================================================================

CREATE TYPE user_role AS ENUM ('super_admin', 'org_admin', 'hr', 'dept_manager', 'employee');
CREATE TYPE attendance_status AS ENUM ('check_in', 'check_out', 'break_start', 'break_end');
CREATE TYPE kiosk_status AS ENUM ('active', 'inactive', 'revoked', 'maintenance');
CREATE TYPE audit_action AS ENUM (
    'user_created', 'user_updated', 'user_deleted', 'user_activated', 'user_deactivated',
    'department_created', 'department_updated', 'department_deleted',
    'kiosk_created', 'kiosk_updated', 'kiosk_revoked', 'kiosk_activated',
    'attendance_log_created', 'attendance_log_updated', 'attendance_log_deleted',
    'admin_login', 'admin_logout', 'permission_granted', 'permission_revoked',
    'export_generated', 'report_generated', 'settings_updated'
);
CREATE TYPE subscription_tier AS ENUM ('free', 'starter', 'professional', 'enterprise');
CREATE TYPE liveness_type AS ENUM ('active', 'passive', 'none');

-- ============================================================================
-- TENANTS TABLE
-- ============================================================================

CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    subscription_tier subscription_tier NOT NULL DEFAULT 'free',
    subscription_expires_at TIMESTAMPTZ,
    max_users INTEGER DEFAULT 10,
    max_kiosks INTEGER DEFAULT 1,
    kiosk_code VARCHAR(10) UNIQUE NOT NULL, -- 10-Digit permanent kiosk code
    sso_provider VARCHAR(50), -- 'saml', 'oidc', 'google', 'azure', etc.
    sso_config JSONB, -- Encrypted SSO configuration
    settings JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tenants_slug ON tenants(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_kiosk_code ON tenants(kiosk_code) WHERE deleted_at IS NULL;

-- ============================================================================
-- USERS TABLE
-- ============================================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id VARCHAR(50) NOT NULL, -- Unique within tenant
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    department_id UUID, -- References departments(id)
    designation VARCHAR(100),
    date_of_joining DATE NOT NULL,
    shift_start_time TIME,
    shift_end_time TIME,
    shift_length_hours DECIMAL(4,2), -- For timeout logic (e.g., 8.0, 9.5)
    role user_role NOT NULL DEFAULT 'employee',
    password_hash VARCHAR(255), -- For non-SSO users
    sso_external_id VARCHAR(255), -- External SSO identifier
    is_active BOOLEAN NOT NULL DEFAULT true,
    data_privacy_consent BOOLEAN NOT NULL DEFAULT false,
    consent_date TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    manager_id UUID REFERENCES users(id) ON DELETE SET NULL,
    employment_type VARCHAR(50) NOT NULL DEFAULT 'full_time',
    work_location VARCHAR(120),
    cost_center VARCHAR(120),
    invite_status VARCHAR(30) NOT NULL DEFAULT 'not_invited',
    invite_sent_at TIMESTAMPTZ,
    offboarded_at TIMESTAMPTZ,
    offboarding_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    -- Ensure unique employee_id per tenant
    CONSTRAINT unique_employee_id_per_tenant UNIQUE (tenant_id, employee_id),
    CONSTRAINT unique_email_per_tenant UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant_id ON users(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_employee_id ON users(employee_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_department_id ON users(department_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_manager_id ON users(manager_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_active ON users(tenant_id, is_active) WHERE deleted_at IS NULL;

-- ============================================================================
-- DEPARTMENTS TABLE
-- ============================================================================

CREATE TABLE departments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50),
    description TEXT,
    manager_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT unique_dept_name_per_tenant UNIQUE (tenant_id, name)
);

CREATE INDEX idx_departments_tenant_id ON departments(tenant_id) WHERE deleted_at IS NULL;

-- Add foreign key constraint for users.department_id
ALTER TABLE users ADD CONSTRAINT fk_users_department 
    FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE SET NULL;

-- ============================================================================
-- EMERGENCY CONTACTS TABLE
-- ============================================================================

CREATE TABLE emergency_contacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    relationship VARCHAR(50),
    phone VARCHAR(20) NOT NULL,
    email VARCHAR(255),
    is_primary BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_emergency_contacts_user_id ON emergency_contacts(user_id);

-- ============================================================================
-- FACE VECTORS TABLE (AES-256 Encrypted Vectors)
-- ============================================================================

CREATE TABLE face_vectors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Encrypted vector stored as BYTEA (AES-256 encrypted)
    encrypted_vector BYTEA NOT NULL,
    -- Embedding model tag (ArcFace-only matching in production)
    model_name VARCHAR(64) NOT NULL DEFAULT 'ArcFace',
    -- Vector metadata (non-sensitive)
    vector_dimension INTEGER NOT NULL DEFAULT 512, -- DeepFace default
    confidence_score DECIMAL(5,4), -- Original match confidence
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_learning_update TIMESTAMPTZ, -- For biometric drift tracking
    -- Ensure one vector per user (can be updated via continuous learning)
    CONSTRAINT unique_user_vector UNIQUE (user_id)
);

-- HNSW Index for fast similarity search (will be created after RLS)
-- Note: We'll create a decrypted vector column for indexing purposes
-- In production, this should be handled via a secure function that decrypts on-the-fly
CREATE INDEX idx_face_vectors_tenant_id ON face_vectors(tenant_id);
CREATE INDEX idx_face_vectors_tenant_model ON face_vectors(tenant_id, model_name);
CREATE INDEX idx_face_vectors_user_id ON face_vectors(user_id);

-- ============================================================================
-- KIOSKS TABLE
-- ============================================================================

CREATE TABLE kiosks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(10) NOT NULL, -- References tenant.kiosk_code or sub-code
    hmac_secret VARCHAR(255) NOT NULL, -- For request signing
    status kiosk_status NOT NULL DEFAULT 'active',
    location TEXT,
    ip_whitelist TEXT[], -- Future: IP restrictions
    geolocation_restrictions JSONB, -- Future: Lat/long boundaries
    last_heartbeat_at TIMESTAMPTZ,
    mqtt_topic VARCHAR(255), -- For IoT door relay
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    CONSTRAINT unique_kiosk_code_per_tenant UNIQUE (tenant_id, code)
);

CREATE INDEX idx_kiosks_tenant_id ON kiosks(tenant_id);
CREATE INDEX idx_kiosks_code ON kiosks(code);
CREATE INDEX idx_kiosks_status ON kiosks(status) WHERE status = 'active';

-- ============================================================================
-- ATTENDANCE LOGS TABLE
-- ============================================================================

CREATE TABLE attendance_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kiosk_id UUID REFERENCES kiosks(id) ON DELETE SET NULL,
    status attendance_status NOT NULL,
    -- Time tracking
    punch_time TIMESTAMPTZ NOT NULL, -- Server-calculated true time
    local_time TIMESTAMPTZ, -- Original local time from kiosk
    monotonic_offset_ms BIGINT, -- Offset for offline time reconciliation
    -- Biometric verification
    face_match_confidence DECIMAL(5,4),
    liveness_type liveness_type,
    liveness_score DECIMAL(5,4),
    verification_method VARCHAR(20) NOT NULL DEFAULT 'biometric', -- 'biometric', 'pin', 'manual'
    -- PIN fallback tracking
    pin_used BOOLEAN DEFAULT false,
    anomaly_detected BOOLEAN DEFAULT false, -- For buddy punching detection
    anomaly_reason TEXT,
    -- Location (future)
    ip_address INET,
    geolocation JSONB,
    -- Metadata
    device_info JSONB,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_attendance_logs_tenant_id ON attendance_logs(tenant_id);
CREATE INDEX idx_attendance_logs_user_id ON attendance_logs(user_id);
CREATE INDEX idx_attendance_logs_punch_time ON attendance_logs(tenant_id, user_id, punch_time DESC);
-- Partial index using non-immutable function removed because Postgres requires
-- index predicates to be IMMUTABLE. Use a regular index or BRIN for date ranges.
CREATE INDEX idx_attendance_logs_date_range ON attendance_logs(tenant_id, punch_time);
CREATE INDEX idx_attendance_logs_kiosk_id ON attendance_logs(kiosk_id);

-- ============================================================================
-- EMPLOYEE ADMINISTRATION & ATTENDANCE OPERATIONS
-- ============================================================================

CREATE TABLE employee_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    document_type VARCHAR(50) NOT NULL,
    name VARCHAR(150) NOT NULL,
    file_url TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::jsonb,
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_employee_documents_user_id ON employee_documents(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_employee_documents_tenant_id ON employee_documents(tenant_id) WHERE deleted_at IS NULL;

CREATE TABLE leave_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    leave_type VARCHAR(50) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    day_count DECIMAL(6,2) NOT NULL DEFAULT 1,
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_leave_requests_tenant_status ON leave_requests(tenant_id, status, start_date DESC);
CREATE INDEX idx_leave_requests_user_id ON leave_requests(user_id, start_date DESC);

CREATE TABLE attendance_regularization_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attendance_log_id UUID REFERENCES attendance_logs(id) ON DELETE SET NULL,
    request_date DATE NOT NULL,
    requested_status attendance_status NOT NULL,
    requested_punch_time TIMESTAMPTZ NOT NULL,
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_regularization_requests_tenant_status ON attendance_regularization_requests(tenant_id, status, request_date DESC);
CREATE INDEX idx_regularization_requests_user_id ON attendance_regularization_requests(user_id, request_date DESC);

CREATE TABLE overtime_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    work_date DATE NOT NULL,
    requested_minutes INTEGER NOT NULL,
    approved_minutes INTEGER NOT NULL DEFAULT 0,
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_overtime_requests_tenant_status ON overtime_requests(tenant_id, status, work_date DESC);
CREATE INDEX idx_overtime_requests_user_id ON overtime_requests(user_id, work_date DESC);

CREATE TABLE shift_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shift_name VARCHAR(120) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    work_days TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    is_rota BOOLEAN NOT NULL DEFAULT false,
    notes TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_shift_assignments_tenant_user ON shift_assignments(tenant_id, user_id, start_date DESC) WHERE deleted_at IS NULL;

CREATE TABLE attendance_exception_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    attendance_log_id UUID NOT NULL REFERENCES attendance_logs(id) ON DELETE CASCADE,
    assigned_to UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    sla_due_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    note TEXT,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_exception_assignment_per_log UNIQUE (attendance_log_id)
);

CREATE INDEX idx_exception_assignments_tenant_status ON attendance_exception_assignments(tenant_id, status, sla_due_at);
CREATE INDEX idx_exception_assignments_assigned_to ON attendance_exception_assignments(assigned_to, status);

CREATE TABLE bulk_change_batches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    change_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'previewed',
    summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    applied_at TIMESTAMPTZ,
    rolled_back_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE bulk_change_batch_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    batch_id UUID NOT NULL REFERENCES bulk_change_batches(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    before_state JSONB NOT NULL,
    after_state JSONB NOT NULL
);

CREATE INDEX idx_bulk_change_batches_tenant_status ON bulk_change_batches(tenant_id, status, created_at DESC);
CREATE INDEX idx_bulk_change_items_batch_id ON bulk_change_batch_items(batch_id);

-- ============================================================================
-- AUDIT LOGS TABLE
-- ============================================================================

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL, -- Admin who performed action
    target_user_id UUID REFERENCES users(id) ON DELETE SET NULL, -- User affected (if applicable)
    action audit_action NOT NULL,
    resource_type VARCHAR(50), -- 'user', 'department', 'kiosk', 'attendance', etc.
    resource_id UUID,
    details JSONB, -- Additional context (e.g., old values, new values)
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);

-- ============================================================================
-- AUTH SESSIONS & MFA CHALLENGES
-- ============================================================================

CREATE TABLE auth_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_jti VARCHAR(128) NOT NULL UNIQUE,
    ip_address INET,
    user_agent TEXT,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_sessions_tenant_user ON auth_sessions(tenant_id, user_id);
CREATE INDEX idx_auth_sessions_active ON auth_sessions(tenant_id, revoked_at, expires_at);

CREATE TABLE auth_mfa_challenges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    code_hash VARCHAR(128) NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_mfa_challenges_user ON auth_mfa_challenges(user_id, expires_at DESC);

-- ============================================================================
-- CUSTOM ROLES & PERMISSIONS
-- ============================================================================

CREATE TABLE custom_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT unique_custom_role_name_per_tenant UNIQUE (tenant_id, name)
);

CREATE INDEX idx_custom_roles_tenant_id ON custom_roles(tenant_id) WHERE deleted_at IS NULL;

CREATE TABLE custom_role_permissions (
    custom_role_id UUID NOT NULL REFERENCES custom_roles(id) ON DELETE CASCADE,
    permission VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (custom_role_id, permission)
);

CREATE TABLE custom_role_assignments (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    custom_role_id UUID NOT NULL REFERENCES custom_roles(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_custom_role_assignments_role_id ON custom_role_assignments(custom_role_id);

-- ============================================================================
-- HRMS WEBHOOK INTEGRATIONS TABLE
-- ============================================================================

CREATE TABLE hrms_integrations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'workday', 'sap', 'bamboo', 'custom'
    webhook_url VARCHAR(500), -- Our endpoint URL
    api_key VARCHAR(255) NOT NULL, -- For authenticating incoming webhooks
    api_secret VARCHAR(255), -- For HMAC signing
    config JSONB DEFAULT '{}'::jsonb, -- Provider-specific configuration
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_provider_per_tenant UNIQUE (tenant_id, provider)
);

CREATE INDEX idx_hrms_integrations_tenant_id ON hrms_integrations(tenant_id);
CREATE INDEX idx_hrms_integrations_api_key ON hrms_integrations(api_key);

-- ============================================================================
-- HRMS SYNC SCHEDULES + LOGS
-- ============================================================================

CREATE TABLE hrms_sync_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    integration_id UUID NOT NULL REFERENCES hrms_integrations(id) ON DELETE CASCADE,
    frequency VARCHAR(20) NOT NULL, -- 'hourly', 'daily', 'weekly', 'monthly'
    day_of_week INT, -- 0=Sunday..6=Saturday for weekly
    time_of_day TIME NOT NULL DEFAULT '00:00',
    timezone VARCHAR(100) NOT NULL DEFAULT 'UTC',
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_hrms_schedule_per_integration UNIQUE (integration_id)
);

CREATE INDEX idx_hrms_sync_schedules_tenant_id ON hrms_sync_schedules(tenant_id);
CREATE INDEX idx_hrms_sync_schedules_integration_id ON hrms_sync_schedules(integration_id);

CREATE TABLE hrms_sync_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    integration_id UUID NOT NULL REFERENCES hrms_integrations(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL, -- 'success', 'failed'
    message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_hrms_sync_logs_tenant_id ON hrms_sync_logs(tenant_id);
CREATE INDEX idx_hrms_sync_logs_integration_id ON hrms_sync_logs(integration_id);
CREATE INDEX idx_hrms_sync_logs_started_at ON hrms_sync_logs(started_at DESC);

-- ============================================================================
-- PAYROLL EXPORT QUEUE TABLE
-- ============================================================================

CREATE TABLE payroll_exports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    export_type VARCHAR(50) NOT NULL, -- 'csv', 'excel', 'pdf', 'api'
    date_range_start DATE NOT NULL,
    date_range_end DATE NOT NULL,
    requested_by UUID NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    file_url VARCHAR(500), -- Supabase Storage URL
    payload JSONB, -- Formatted timesheet data
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_payroll_exports_tenant_id ON payroll_exports(tenant_id);
CREATE INDEX idx_payroll_exports_status ON payroll_exports(status);

-- ============================================================================
-- REPORT SCHEDULES + DELIVERY LOGS
-- ============================================================================

CREATE TABLE report_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    report_type VARCHAR(50) NOT NULL, -- 'attendance'
    name VARCHAR(120),
    frequency VARCHAR(20) NOT NULL, -- 'daily','weekly','monthly'
    day_of_week INT, -- 0=Sunday..6=Saturday (weekly)
    time_of_day TIME NOT NULL DEFAULT '08:00',
    timezone VARCHAR(100) NOT NULL DEFAULT 'UTC',
    recipients TEXT[] NOT NULL,
    filters JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_report_schedules_tenant_id ON report_schedules(tenant_id);
CREATE INDEX idx_report_schedules_report_type ON report_schedules(report_type);

CREATE TABLE report_delivery_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    schedule_id UUID REFERENCES report_schedules(id) ON DELETE SET NULL,
    report_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL, -- 'queued','sent','failed'
    message TEXT,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_report_delivery_logs_tenant_id ON report_delivery_logs(tenant_id);
CREATE INDEX idx_report_delivery_logs_schedule_id ON report_delivery_logs(schedule_id);
CREATE INDEX idx_report_delivery_logs_delivered_at ON report_delivery_logs(delivered_at DESC);

-- ============================================================================
-- OFFLINE QUEUE TABLE (for IndexedDB sync)
-- ============================================================================

CREATE TABLE offline_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kiosk_id UUID NOT NULL REFERENCES kiosks(id) ON DELETE CASCADE,
    -- Encrypted payload from IndexedDB
    encrypted_payload BYTEA NOT NULL,
    public_key_fingerprint VARCHAR(64), -- To identify which public key was used
    sync_status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'synced', 'failed'
    retry_count INTEGER DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMPTZ
);

CREATE INDEX idx_offline_queue_tenant_id ON offline_queue(tenant_id);
CREATE INDEX idx_offline_queue_sync_status ON offline_queue(sync_status) WHERE sync_status = 'pending';

-- ============================================================================
-- ROW-LEVEL SECURITY (RLS) POLICIES
-- ============================================================================

-- Enable RLS on all tables
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE departments ENABLE ROW LEVEL SECURITY;
ALTER TABLE emergency_contacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE face_vectors ENABLE ROW LEVEL SECURITY;
ALTER TABLE kiosks ENABLE ROW LEVEL SECURITY;
ALTER TABLE attendance_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE hrms_integrations ENABLE ROW LEVEL SECURITY;
ALTER TABLE hrms_sync_schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE hrms_sync_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE payroll_exports ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_delivery_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE offline_queue ENABLE ROW LEVEL SECURITY;

-- ============================================================================
-- RLS POLICY FUNCTIONS
-- ============================================================================

-- Function to get current user's tenant_id from JWT
-- In Supabase, this would use auth.uid() or custom claims
-- For now, we'll create a helper function that can be called by the application
CREATE OR REPLACE FUNCTION get_user_tenant_id()
RETURNS UUID AS $$
BEGIN
    -- This will be set by the application via SET LOCAL
    -- For Supabase, use: current_setting('app.current_tenant_id', true)::UUID
    RETURN current_setting('app.current_tenant_id', true)::UUID;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to check if user is super admin
CREATE OR REPLACE FUNCTION is_super_admin()
RETURNS BOOLEAN AS $$
BEGIN
    -- Super admins have a special claim in JWT
    RETURN current_setting('app.is_super_admin', true)::BOOLEAN = true;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- TENANTS RLS POLICIES
-- ============================================================================

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
-- USERS RLS POLICIES
-- ============================================================================

-- Super admins can read all users (but not PII - handled in application layer)
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
    USING (id = current_setting('app.current_user_id', true)::UUID);

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
-- DEPARTMENTS RLS POLICIES
-- ============================================================================

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
-- EMERGENCY CONTACTS RLS POLICIES
-- ============================================================================

CREATE POLICY "users_read_own_contacts" ON emergency_contacts
    FOR SELECT
    USING (
        user_id = current_setting('app.current_user_id', true)::UUID
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
-- FACE VECTORS RLS POLICIES
-- ============================================================================

-- Only AI service and authorized users can read face vectors
CREATE POLICY "tenant_face_vectors_read" ON face_vectors
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

-- Only AI service can insert/update face vectors (via service role)
-- Application will bypass RLS using service role for AI operations
CREATE POLICY "ai_service_manage_vectors" ON face_vectors
    FOR ALL
    USING (current_setting('app.is_ai_service', true)::BOOLEAN = true);

-- ============================================================================
-- KIOSKS RLS POLICIES
-- ============================================================================

CREATE POLICY "tenant_kiosks_read" ON kiosks
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "admin_insert_kiosks" ON kiosks
    FOR INSERT
    WITH CHECK (tenant_id = get_user_tenant_id());

CREATE POLICY "admin_update_kiosks" ON kiosks
    FOR UPDATE
    USING (tenant_id = get_user_tenant_id());

-- Kiosks can read their own record via HMAC (handled in application layer)

-- ============================================================================
-- ATTENDANCE LOGS RLS POLICIES
-- ============================================================================

CREATE POLICY "tenant_attendance_read" ON attendance_logs
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "users_read_own_attendance" ON attendance_logs
    FOR SELECT
    USING (user_id = current_setting('app.current_user_id', true)::UUID);

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
-- AUDIT LOGS RLS POLICIES
-- ============================================================================

CREATE POLICY "tenant_audit_read" ON audit_logs
    FOR SELECT
    USING (tenant_id = get_user_tenant_id() OR tenant_id IS NULL);

-- Audit logs are insert-only (no updates/deletes)
CREATE POLICY "system_insert_audit" ON audit_logs
    FOR INSERT
    WITH CHECK (true); -- Application layer enforces tenant_id

-- ============================================================================
-- HRMS INTEGRATIONS RLS POLICIES
-- ============================================================================

CREATE POLICY "tenant_hrms_read" ON hrms_integrations
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "admin_manage_hrms" ON hrms_integrations
    FOR ALL
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "tenant_hrms_sync_read" ON hrms_sync_schedules
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "admin_manage_hrms_sync" ON hrms_sync_schedules
    FOR ALL
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "tenant_hrms_sync_logs_read" ON hrms_sync_logs
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "admin_manage_hrms_sync_logs" ON hrms_sync_logs
    FOR ALL
    USING (tenant_id = get_user_tenant_id());

-- ============================================================================
-- PAYROLL EXPORTS RLS POLICIES
-- ============================================================================

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
-- REPORT SCHEDULES + DELIVERY LOGS RLS POLICIES
-- ============================================================================

CREATE POLICY "tenant_report_schedule_read" ON report_schedules
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "admin_manage_report_schedule" ON report_schedules
    FOR ALL
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "tenant_report_delivery_read" ON report_delivery_logs
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

CREATE POLICY "admin_manage_report_delivery" ON report_delivery_logs
    FOR ALL
    USING (tenant_id = get_user_tenant_id());

-- ============================================================================
-- OFFLINE QUEUE RLS POLICIES
-- ============================================================================

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
-- AUTOMATED DATA PURGING FUNCTION (GDPR/CCPA Compliance)
-- ============================================================================

CREATE OR REPLACE FUNCTION purge_terminated_employee_data()
RETURNS void AS $$
BEGIN
    -- Hard delete face vectors for users deleted more than 30 days ago
    DELETE FROM face_vectors
    WHERE user_id IN (
        SELECT id FROM users
        WHERE deleted_at IS NOT NULL
        AND deleted_at < NOW() - INTERVAL '30 days'
    );
    
    -- Log the purge action
    INSERT INTO audit_logs (tenant_id, action, resource_type, details)
    SELECT 
        tenant_id,
        'user_deleted'::audit_action,
        'face_vector',
        jsonb_build_object('purged_count', COUNT(*))
    FROM face_vectors
    WHERE user_id IN (
        SELECT id FROM users
        WHERE deleted_at IS NOT NULL
        AND deleted_at < NOW() - INTERVAL '30 days'
    )
    GROUP BY tenant_id;
END;
$$ LANGUAGE plpgsql;

-- Schedule this function to run daily (requires pg_cron extension)
-- CREATE EXTENSION IF NOT EXISTS pg_cron;
-- SELECT cron.schedule('purge-terminated-employees', '0 2 * * *', 'SELECT purge_terminated_employee_data();');

-- ============================================================================
-- CONTINUOUS LEARNING UPDATE FUNCTION (Biometric Drift Fix)
-- ============================================================================

CREATE OR REPLACE FUNCTION update_face_vector_learning(
    p_user_id UUID,
    p_new_confidence DECIMAL,
    p_encrypted_vector BYTEA
)
RETURNS void AS $$
DECLARE
    v_last_update TIMESTAMPTZ;
    v_threshold DECIMAL := 0.98; -- 98% confidence threshold
    v_learning_rate DECIMAL := 0.05; -- 5% blend
    v_max_frequency_days INTEGER := 7; -- Max once per week
BEGIN
    -- Check if confidence meets threshold
    IF p_new_confidence < v_threshold THEN
        RETURN;
    END IF;
    
    -- Check last update time (max once per week)
    SELECT last_learning_update INTO v_last_update
    FROM face_vectors
    WHERE user_id = p_user_id;
    
    IF v_last_update IS NOT NULL AND v_last_update > NOW() - (v_max_frequency_days || ' days')::INTERVAL THEN
        RETURN; -- Too soon, skip update
    END IF;
    
    -- Update vector with blended learning (95% old, 5% new)
    -- Note: Actual vector blending happens in Python AI service
    -- This function just updates the timestamp
    UPDATE face_vectors
    SET 
        last_learning_update = NOW(),
        updated_at = NOW(),
        confidence_score = GREATEST(confidence_score, p_new_confidence)
    WHERE user_id = p_user_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- TRIGGERS FOR UPDATED_AT
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_departments_updated_at BEFORE UPDATE ON departments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_emergency_contacts_updated_at BEFORE UPDATE ON emergency_contacts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_face_vectors_updated_at BEFORE UPDATE ON face_vectors
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_kiosks_updated_at BEFORE UPDATE ON kiosks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_attendance_logs_updated_at BEFORE UPDATE ON attendance_logs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_hrms_integrations_updated_at BEFORE UPDATE ON hrms_integrations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_hrms_sync_schedules_updated_at BEFORE UPDATE ON hrms_sync_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_report_schedules_updated_at BEFORE UPDATE ON report_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- AUDIT LOG TRIGGER FUNCTION
-- ============================================================================

CREATE OR REPLACE FUNCTION log_user_changes()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        INSERT INTO audit_logs (tenant_id, user_id, target_user_id, action, resource_type, resource_id, details)
        VALUES (
            OLD.tenant_id,
            current_setting('app.current_user_id', true)::UUID,
            OLD.id,
            'user_deleted'::audit_action,
            'user',
            OLD.id,
            jsonb_build_object('employee_id', OLD.employee_id, 'email', OLD.email)
        );
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        -- Log significant changes
        IF OLD.is_active != NEW.is_active THEN
            INSERT INTO audit_logs (tenant_id, user_id, target_user_id, action, resource_type, resource_id, details)
            VALUES (
                NEW.tenant_id,
                current_setting('app.current_user_id', true)::UUID,
                NEW.id,
                CASE WHEN NEW.is_active THEN 'user_activated'::audit_action ELSE 'user_deactivated'::audit_action END,
                'user',
                NEW.id,
                jsonb_build_object('employee_id', NEW.employee_id)
            );
        END IF;
        
        IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
            INSERT INTO audit_logs (tenant_id, user_id, target_user_id, action, resource_type, resource_id, details)
            VALUES (
                NEW.tenant_id,
                current_setting('app.current_user_id', true)::UUID,
                NEW.id,
                'user_deleted'::audit_action,
                'user',
                NEW.id,
                jsonb_build_object('employee_id', NEW.employee_id)
            );
        END IF;
        
        RETURN NEW;
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO audit_logs (tenant_id, user_id, target_user_id, action, resource_type, resource_id, details)
        VALUES (
            NEW.tenant_id,
            current_setting('app.current_user_id', true)::UUID,
            NEW.id,
            'user_created'::audit_action,
            'user',
            NEW.id,
            jsonb_build_object('employee_id', NEW.employee_id, 'email', NEW.email)
        );
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_user_changes
    AFTER INSERT OR UPDATE OR DELETE ON users
    FOR EACH ROW EXECUTE FUNCTION log_user_changes();

-- ============================================================================
-- INITIAL DATA / SEEDING (Optional)
-- ============================================================================

-- Create a default super admin tenant (for testing)
-- In production, this should be done via application onboarding
-- INSERT INTO tenants (name, slug, subscription_tier, kiosk_code)
-- VALUES ('Super Admin Org', 'super-admin', 'enterprise', '0000000000')
-- ON CONFLICT DO NOTHING;

-- ============================================================================
-- NOTES FOR PRODUCTION DEPLOYMENT
-- ============================================================================

-- 1. HNSW Index Creation:
--    After initial data load, create HNSW index on decrypted vectors:
--    CREATE INDEX ON face_vectors USING hnsw (decrypted_vector vector_cosine_ops)
--    WITH (m = 16, ef_construction = 64);
--    Note: This requires a decrypted vector column or a secure function that decrypts on-the-fly

-- 2. Supabase Configuration:
--    - Enable RLS on all tables (already done)
--    - Configure JWT claims to include tenant_id and user_id
--    - Set up service role for AI microservice
--    - Configure storage buckets for encrypted file hosting

-- 3. Security:
--    - All face vectors must be AES-256 encrypted before storage
--    - Application must set 'app.current_tenant_id' and 'app.current_user_id' via SET LOCAL
--    - HMAC secrets for kiosks must be generated securely (use crypto.randomBytes)

-- 4. Performance:
--    - Monitor HNSW index performance
--    - Add additional indexes based on query patterns
--    - Consider partitioning attendance_logs by date for large tenants

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================
