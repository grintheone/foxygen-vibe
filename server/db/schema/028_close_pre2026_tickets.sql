-- Bootstrap replays schema files. Already closed tickets are left unchanged.
-- Keep the original rows for recovery without changing reports or signatures.
CREATE TABLE IF NOT EXISTS ticket_pre2026_closure_backup (
  ticket_id UUID PRIMARY KEY,
  snapshot JSONB NOT NULL,
  backed_up_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

WITH targets AS MATERIALIZED (
  SELECT *
  FROM tickets
  WHERE created_at < TIMESTAMP '2026-01-01 00:00:00'
    AND status IS DISTINCT FROM 'closed'
  FOR UPDATE
), backed_up AS (
  INSERT INTO ticket_pre2026_closure_backup (ticket_id, snapshot)
  SELECT id, to_jsonb(targets) FROM targets
  ON CONFLICT (ticket_id) DO NOTHING
)
UPDATE tickets AS t
SET status = 'closed',
    closed_at = COALESCE(t.closed_at, NOW() AT TIME ZONE 'UTC'),
    app_version = t.app_version + 1
FROM targets
WHERE t.id = targets.id;
