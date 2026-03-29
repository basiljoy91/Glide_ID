CREATE TABLE IF NOT EXISTS email_verification_challenges (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  email VARCHAR(255) NOT NULL,
  scope VARCHAR(100) NOT NULL,
  code_hash VARCHAR(128) NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  verified_at TIMESTAMPTZ,
  consumed_at TIMESTAMPTZ,
  ip_address INET,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_verification_challenges_email_scope
  ON email_verification_challenges(email, scope, expires_at DESC);
