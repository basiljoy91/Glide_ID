-- Kiosk fleet telemetry, incidents, commands, and report saved views.

CREATE TABLE IF NOT EXISTS kiosk_telemetry_samples (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kiosk_id UUID NOT NULL REFERENCES kiosks(id) ON DELETE CASCADE,
    status kiosk_status NOT NULL DEFAULT 'active',
    app_version VARCHAR(50),
    os_version VARCHAR(100),
    battery_percent INTEGER,
    network_strength INTEGER,
    storage_free_mb INTEGER,
    storage_total_mb INTEGER,
    memory_usage_percent INTEGER,
    metadata JSONB DEFAULT '{}'::jsonb,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_kiosk_telemetry_samples_tenant_id ON kiosk_telemetry_samples(tenant_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_kiosk_telemetry_samples_kiosk_id ON kiosk_telemetry_samples(kiosk_id, recorded_at DESC);

CREATE TABLE IF NOT EXISTS kiosk_incidents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kiosk_id UUID NOT NULL REFERENCES kiosks(id) ON DELETE CASCADE,
    incident_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning',
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    title VARCHAR(150) NOT NULL,
    details TEXT,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by UUID REFERENCES users(id) ON DELETE SET NULL,
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_kiosk_incidents_tenant_id ON kiosk_incidents(tenant_id, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_kiosk_incidents_kiosk_id ON kiosk_incidents(kiosk_id, status);

CREATE TABLE IF NOT EXISTS kiosk_commands (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kiosk_id UUID NOT NULL REFERENCES kiosks(id) ON DELETE CASCADE,
    command_type VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    payload JSONB DEFAULT '{}'::jsonb,
    requested_by UUID REFERENCES users(id) ON DELETE SET NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    last_error TEXT
);

CREATE INDEX IF NOT EXISTS idx_kiosk_commands_tenant_id ON kiosk_commands(tenant_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_kiosk_commands_kiosk_id ON kiosk_commands(kiosk_id, status);

CREATE TABLE IF NOT EXISTS report_saved_views (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    report_type VARCHAR(50) NOT NULL,
    name VARCHAR(120) NOT NULL,
    filters JSONB DEFAULT '{}'::jsonb,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_report_saved_views_tenant_id ON report_saved_views(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_report_saved_views_report_type ON report_saved_views(report_type);

ALTER TABLE kiosk_telemetry_samples ENABLE ROW LEVEL SECURITY;
ALTER TABLE kiosk_incidents ENABLE ROW LEVEL SECURITY;
ALTER TABLE kiosk_commands ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_saved_views ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "tenant_kiosk_telemetry_read" ON kiosk_telemetry_samples;
CREATE POLICY "tenant_kiosk_telemetry_read" ON kiosk_telemetry_samples
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "admin_manage_kiosk_telemetry" ON kiosk_telemetry_samples;
CREATE POLICY "admin_manage_kiosk_telemetry" ON kiosk_telemetry_samples
    FOR ALL
    USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "tenant_kiosk_incidents_read" ON kiosk_incidents;
CREATE POLICY "tenant_kiosk_incidents_read" ON kiosk_incidents
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "admin_manage_kiosk_incidents" ON kiosk_incidents;
CREATE POLICY "admin_manage_kiosk_incidents" ON kiosk_incidents
    FOR ALL
    USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "tenant_kiosk_commands_read" ON kiosk_commands;
CREATE POLICY "tenant_kiosk_commands_read" ON kiosk_commands
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "admin_manage_kiosk_commands" ON kiosk_commands;
CREATE POLICY "admin_manage_kiosk_commands" ON kiosk_commands
    FOR ALL
    USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "tenant_report_saved_views_read" ON report_saved_views;
CREATE POLICY "tenant_report_saved_views_read" ON report_saved_views
    FOR SELECT
    USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "admin_manage_report_saved_views" ON report_saved_views;
CREATE POLICY "admin_manage_report_saved_views" ON report_saved_views
    FOR ALL
    USING (tenant_id = get_user_tenant_id());
