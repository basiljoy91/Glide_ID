-- 004_billing_core.sql
-- Adds core billing tables for super admin billing, subscriptions, and invoice tracking.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'billing_subscription_status') THEN
    CREATE TYPE billing_subscription_status AS ENUM ('trialing', 'active', 'past_due', 'canceled');
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'billing_invoice_status') THEN
    CREATE TYPE billing_invoice_status AS ENUM ('draft', 'open', 'paid', 'void', 'uncollectible');
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS billing_subscriptions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  plan_tier subscription_tier NOT NULL DEFAULT 'free',
  status billing_subscription_status NOT NULL DEFAULT 'active',
  billing_cycle VARCHAR(20) NOT NULL DEFAULT 'monthly',
  seat_count INTEGER NOT NULL DEFAULT 1 CHECK (seat_count >= 1),
  base_amount_cents INTEGER NOT NULL DEFAULT 0 CHECK (base_amount_cents >= 0),
  per_seat_amount_cents INTEGER NOT NULL DEFAULT 0 CHECK (per_seat_amount_cents >= 0),
  provider VARCHAR(50),
  provider_customer_id VARCHAR(255),
  provider_subscription_id VARCHAR(255),
  current_period_start TIMESTAMPTZ,
  current_period_end TIMESTAMPTZ,
  next_invoice_at TIMESTAMPTZ,
  canceled_at TIMESTAMPTZ,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT unique_billing_subscription_per_tenant UNIQUE (tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_billing_subscriptions_tenant_id ON billing_subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_billing_subscriptions_status ON billing_subscriptions(status);

CREATE TABLE IF NOT EXISTS billing_invoices (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  subscription_id UUID REFERENCES billing_subscriptions(id) ON DELETE SET NULL,
  invoice_number VARCHAR(64) NOT NULL UNIQUE,
  status billing_invoice_status NOT NULL DEFAULT 'draft',
  period_start DATE NOT NULL,
  period_end DATE NOT NULL,
  subtotal_cents INTEGER NOT NULL DEFAULT 0 CHECK (subtotal_cents >= 0),
  tax_cents INTEGER NOT NULL DEFAULT 0 CHECK (tax_cents >= 0),
  total_cents INTEGER NOT NULL DEFAULT 0 CHECK (total_cents >= 0),
  due_at TIMESTAMPTZ,
  paid_at TIMESTAMPTZ,
  payment_reference VARCHAR(255),
  notes TEXT,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_billing_invoices_tenant_id ON billing_invoices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_billing_invoices_status ON billing_invoices(status);
CREATE INDEX IF NOT EXISTS idx_billing_invoices_created_at ON billing_invoices(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_billing_invoices_period ON billing_invoices(period_start, period_end);

-- Backfill billing subscriptions from existing tenants if not present.
INSERT INTO billing_subscriptions (
  tenant_id,
  plan_tier,
  status,
  billing_cycle,
  seat_count,
  base_amount_cents,
  per_seat_amount_cents,
  current_period_start,
  current_period_end,
  next_invoice_at,
  created_at,
  updated_at
)
SELECT
  t.id,
  t.subscription_tier,
  CASE WHEN t.deleted_at IS NULL THEN 'active'::billing_subscription_status ELSE 'canceled'::billing_subscription_status END,
  'monthly',
  GREATEST(t.max_users, 1),
  CASE t.subscription_tier
    WHEN 'free' THEN 0
    WHEN 'starter' THEN 19900
    WHEN 'professional' THEN 49900
    WHEN 'enterprise' THEN 99900
    ELSE 0
  END,
  0,
  NOW() - INTERVAL '30 days',
  NOW(),
  NOW() + INTERVAL '30 days',
  NOW(),
  NOW()
FROM tenants t
ON CONFLICT (tenant_id) DO NOTHING;
