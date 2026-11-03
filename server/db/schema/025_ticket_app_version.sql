-- Version of the latest app write, echoed by the legacy integration as appVersion.
-- Existing legacy tickets start at zero; existing closed tickets are protected.
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS app_version BIGINT NOT NULL DEFAULT 0
  CHECK (app_version >= 0);
UPDATE tickets SET app_version = 1 WHERE status = 'closed' AND app_version = 0;
