CREATE TABLE IF NOT EXISTS hrms_webhook_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  integration_id UUID REFERENCES hrms_integrations(id) ON DELETE SET NULL,
  provider VARCHAR(50) NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  signature TEXT,
  status VARCHAR(20) NOT NULL DEFAULT 'received',
  retry_count INTEGER NOT NULL DEFAULT 0,
  error_message TEXT,
  next_retry_at TIMESTAMPTZ,
  processed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_hrms_webhook_events_tenant_id ON hrms_webhook_events(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_hrms_webhook_events_integration_id ON hrms_webhook_events(integration_id, status);

CREATE TABLE IF NOT EXISTS hrms_sync_conflicts (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  integration_id UUID NOT NULL REFERENCES hrms_integrations(id) ON DELETE CASCADE,
  external_record_id VARCHAR(255) NOT NULL,
  field_name VARCHAR(100) NOT NULL,
  local_value JSONB,
  external_value JSONB,
  status VARCHAR(30) NOT NULL DEFAULT 'open',
  resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
  resolved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_hrms_sync_conflicts_tenant_id ON hrms_sync_conflicts(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_hrms_sync_conflicts_integration_id ON hrms_sync_conflicts(integration_id, status);

CREATE TABLE IF NOT EXISTS support_tickets (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  submitted_by UUID REFERENCES users(id) ON DELETE SET NULL,
  category VARCHAR(50) NOT NULL DEFAULT 'general',
  priority VARCHAR(20) NOT NULL DEFAULT 'normal',
  subject VARCHAR(150) NOT NULL,
  description TEXT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_support_tickets_tenant_id ON support_tickets(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS org_notifications (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  notification_type VARCHAR(80) NOT NULL,
  title VARCHAR(150) NOT NULL,
  body TEXT NOT NULL,
  severity VARCHAR(20) NOT NULL DEFAULT 'info',
  is_read BOOLEAN NOT NULL DEFAULT false,
  action_url VARCHAR(255),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  read_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_org_notifications_tenant_id ON org_notifications(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_org_notifications_user_id ON org_notifications(user_id, is_read);

ALTER TABLE hrms_webhook_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE hrms_sync_conflicts ENABLE ROW LEVEL SECURITY;
ALTER TABLE support_tickets ENABLE ROW LEVEL SECURITY;
ALTER TABLE org_notifications ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "tenant_hrms_webhook_events_read" ON hrms_webhook_events;
CREATE POLICY "tenant_hrms_webhook_events_read" ON hrms_webhook_events
  FOR SELECT
  USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "admin_manage_hrms_webhook_events" ON hrms_webhook_events;
CREATE POLICY "admin_manage_hrms_webhook_events" ON hrms_webhook_events
  FOR ALL
  USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "tenant_hrms_sync_conflicts_read" ON hrms_sync_conflicts;
CREATE POLICY "tenant_hrms_sync_conflicts_read" ON hrms_sync_conflicts
  FOR SELECT
  USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "admin_manage_hrms_sync_conflicts" ON hrms_sync_conflicts;
CREATE POLICY "admin_manage_hrms_sync_conflicts" ON hrms_sync_conflicts
  FOR ALL
  USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "tenant_support_tickets_read" ON support_tickets;
CREATE POLICY "tenant_support_tickets_read" ON support_tickets
  FOR SELECT
  USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "tenant_support_tickets_manage" ON support_tickets;
CREATE POLICY "tenant_support_tickets_manage" ON support_tickets
  FOR ALL
  USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "tenant_org_notifications_read" ON org_notifications;
CREATE POLICY "tenant_org_notifications_read" ON org_notifications
  FOR SELECT
  USING (tenant_id = get_user_tenant_id());

DROP POLICY IF EXISTS "tenant_org_notifications_manage" ON org_notifications;
CREATE POLICY "tenant_org_notifications_manage" ON org_notifications
  FOR ALL
  USING (tenant_id = get_user_tenant_id());
