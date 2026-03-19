ALTER TABLE tickets
ADD COLUMN IF NOT EXISTS sync_source TEXT DEFAULT NULL;

ALTER TABLE tickets
ADD COLUMN IF NOT EXISTS sync_key TEXT DEFAULT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS tickets_sync_source_key_idx
ON tickets (sync_source, sync_key)
WHERE sync_source IS NOT NULL AND sync_key IS NOT NULL;
