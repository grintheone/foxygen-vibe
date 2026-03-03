CREATE TABLE IF NOT EXISTS regions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL UNIQUE
);

ALTER TABLE regions
ALTER COLUMN id SET DEFAULT gen_random_uuid();

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'regions_title_key'
  ) THEN
    ALTER TABLE regions
    ADD CONSTRAINT regions_title_key UNIQUE (title);
  END IF;
END $$;
