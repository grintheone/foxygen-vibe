CREATE TABLE IF NOT EXISTS refresh_tokens (
  token_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES accounts(user_id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  rotated_at TIMESTAMPTZ,
  replaced_by UUID REFERENCES refresh_tokens(token_id) ON DELETE SET NULL,
  revoked_at TIMESTAMPTZ
);
