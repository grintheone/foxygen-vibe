-- App-facing reads share this rule, including totals and archive facets.
-- Keep the stored tickets intact for imports, synchronization, and reactivation.
CREATE OR REPLACE VIEW visible_tickets AS
SELECT t.*
FROM tickets t
WHERE NOT EXISTS (
  SELECT 1
  FROM accounts a
  WHERE a.user_id = t.executor
    AND a.disabled = TRUE
);
