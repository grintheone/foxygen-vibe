-- Existing/imported files have no digest; new uploads are unique by bytes per ticket.
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS content_sha256 TEXT;
CREATE INDEX IF NOT EXISTS attachments_ref_id_idx ON attachments (ref_id);
CREATE UNIQUE INDEX IF NOT EXISTS attachments_ticket_content_sha256_idx
  ON attachments (ref_id, content_sha256) WHERE content_sha256 IS NOT NULL;

-- Keep delivery receipts across API restarts and attachment deletion/re-upload.
CREATE TABLE IF NOT EXISTS ticket_attachment_deliveries (
  ticket_id UUID NOT NULL,
  attachment_id TEXT NOT NULL,
  delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (ticket_id, attachment_id)
);
