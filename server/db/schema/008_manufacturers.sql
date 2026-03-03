CREATE TABLE IF NOT EXISTS manufacturers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL DEFAULT ''
);

ALTER TABLE manufacturers
ALTER COLUMN id SET DEFAULT gen_random_uuid();

ALTER TABLE manufacturers
ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '';
