CREATE TABLE IF NOT EXISTS research_type (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL UNIQUE
);

ALTER TABLE research_type
ALTER COLUMN id SET DEFAULT gen_random_uuid();

ALTER TABLE research_type
ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '';
