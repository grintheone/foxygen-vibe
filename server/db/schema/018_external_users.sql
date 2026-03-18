CREATE TABLE IF NOT EXISTS external_users (
  id UUID PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  linked_user_id UUID REFERENCES accounts(user_id) ON DELETE SET NULL
);

ALTER TABLE external_users
ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '';

ALTER TABLE external_users
ADD COLUMN IF NOT EXISTS linked_user_id UUID REFERENCES accounts(user_id) ON DELETE SET NULL;
