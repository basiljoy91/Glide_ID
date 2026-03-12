-- Add HRMS sync schedules/logs and report schedules/logs

CREATE TABLE IF NOT EXISTS hrms_sync_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    integration_id UUID NOT NULL REFERENCES hrms_integrations(id) ON DELETE CASCADE,
    frequency VARCHAR(20) NOT NULL,
    day_of_week INT,
    time_of_day TIME NOT NULL DEFAULT '00:00',
    timezone VARCHAR(100) NOT NULL DEFAULT 'UTC',
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_hrms_schedule_per_integration UNIQUE (integration_id)
);

CREATE INDEX IF NOT EXISTS idx_hrms_sync_schedules_tenant_id ON hrms_sync_schedules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_hrms_sync_schedules_integration_id ON hrms_sync_schedules(integration_id);

CREATE TABLE IF NOT EXISTS hrms_sync_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    integration_id UUID NOT NULL REFERENCES hrms_integrations(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_hrms_sync_logs_tenant_id ON hrms_sync_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_hrms_sync_logs_integration_id ON hrms_sync_logs(integration_id);
CREATE INDEX IF NOT EXISTS idx_hrms_sync_logs_started_at ON hrms_sync_logs(started_at DESC);

CREATE TABLE IF NOT EXISTS report_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    report_type VARCHAR(50) NOT NULL,
    name VARCHAR(120),
    frequency VARCHAR(20) NOT NULL,
    day_of_week INT,
    time_of_day TIME NOT NULL DEFAULT '08:00',
    timezone VARCHAR(100) NOT NULL DEFAULT 'UTC',
    recipients TEXT[] NOT NULL,
    filters JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_report_schedules_tenant_id ON report_schedules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_report_schedules_report_type ON report_schedules(report_type);

CREATE TABLE IF NOT EXISTS report_delivery_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    schedule_id UUID REFERENCES report_schedules(id) ON DELETE SET NULL,
    report_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    message TEXT,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_report_delivery_logs_tenant_id ON report_delivery_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_report_delivery_logs_schedule_id ON report_delivery_logs(schedule_id);
CREATE INDEX IF NOT EXISTS idx_report_delivery_logs_delivered_at ON report_delivery_logs(delivered_at DESC);

ALTER TABLE hrms_sync_schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE hrms_sync_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_delivery_logs ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "tenant_hrms_sync_read" ON hrms_sync_schedules;
DROP POLICY IF EXISTS "admin_manage_hrms_sync" ON hrms_sync_schedules;
DROP POLICY IF EXISTS "tenant_hrms_sync_logs_read" ON hrms_sync_logs;
DROP POLICY IF EXISTS "admin_manage_hrms_sync_logs" ON hrms_sync_logs;
DROP POLICY IF EXISTS "tenant_report_schedule_read" ON report_schedules;
DROP POLICY IF EXISTS "admin_manage_report_schedule" ON report_schedules;
DROP POLICY IF EXISTS "tenant_report_delivery_read" ON report_delivery_logs;
DROP POLICY IF EXISTS "admin_manage_report_delivery" ON report_delivery_logs;

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

CREATE TRIGGER update_hrms_sync_schedules_updated_at BEFORE UPDATE ON hrms_sync_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_report_schedules_updated_at BEFORE UPDATE ON report_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
