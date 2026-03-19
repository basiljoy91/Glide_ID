-- ============================================================================
-- Migration: Org Settings, Sessions, MFA, and Custom Roles
-- Description: Adds auth session tracking, MFA challenges, and custom role tables
-- Date: 2026-03-19
-- ============================================================================

CREATE TABLE IF NOT EXISTS auth_sessions (
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

CREATE INDEX IF NOT EXISTS idx_auth_sessions_tenant_user ON auth_sessions(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_active ON auth_sessions(tenant_id, revoked_at, expires_at);

CREATE TABLE IF NOT EXISTS auth_mfa_challenges (
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

CREATE INDEX IF NOT EXISTS idx_auth_mfa_challenges_user ON auth_mfa_challenges(user_id, expires_at DESC);

CREATE TABLE IF NOT EXISTS custom_roles (
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

CREATE INDEX IF NOT EXISTS idx_custom_roles_tenant_id ON custom_roles(tenant_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS custom_role_permissions (
    custom_role_id UUID NOT NULL REFERENCES custom_roles(id) ON DELETE CASCADE,
    permission VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (custom_role_id, permission)
);

CREATE TABLE IF NOT EXISTS custom_role_assignments (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    custom_role_id UUID NOT NULL REFERENCES custom_roles(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_role_assignments_role_id ON custom_role_assignments(custom_role_id);
