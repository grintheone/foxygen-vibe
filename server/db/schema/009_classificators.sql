CREATE TABLE IF NOT EXISTS classificators (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL DEFAULT '',
  manufacturer UUID REFERENCES manufacturers(id) ON DELETE SET NULL,
  research_type UUID REFERENCES research_type(id) ON DELETE SET NULL,
  registration_certificate JSONB NOT NULL DEFAULT '{}',
  maintenance_regulations JSONB NOT NULL DEFAULT '{}',
  attachments TEXT[] NOT NULL DEFAULT '{}',
  images TEXT[] NOT NULL DEFAULT '{}'
);

ALTER TABLE classificators
ALTER COLUMN id SET DEFAULT gen_random_uuid();

ALTER TABLE classificators
ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '';

ALTER TABLE classificators
ADD COLUMN IF NOT EXISTS manufacturer UUID REFERENCES manufacturers(id) ON DELETE SET NULL;

ALTER TABLE classificators
ADD COLUMN IF NOT EXISTS research_type UUID REFERENCES research_type(id) ON DELETE SET NULL;

ALTER TABLE classificators
ADD COLUMN IF NOT EXISTS registration_certificate JSONB NOT NULL DEFAULT '{}';

ALTER TABLE classificators
ADD COLUMN IF NOT EXISTS maintenance_regulations JSONB NOT NULL DEFAULT '{}';

ALTER TABLE classificators
ADD COLUMN IF NOT EXISTS attachments TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE classificators
ADD COLUMN IF NOT EXISTS images TEXT[] NOT NULL DEFAULT '{}';
