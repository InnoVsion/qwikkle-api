-- +goose Up
ALTER TABLE users
	ADD COLUMN IF NOT EXISTS gender text,
	ADD COLUMN IF NOT EXISTS date_of_birth date,
	ADD COLUMN IF NOT EXISTS country text,
	ADD COLUMN IF NOT EXISTS interests text[] NOT NULL DEFAULT '{}',
	ADD COLUMN IF NOT EXISTS avatar_storage_key text,
	ADD COLUMN IF NOT EXISTS avatar_download_url text;

CREATE INDEX IF NOT EXISTS users_country_idx ON users (country);

ALTER TABLE organization_documents
	ADD COLUMN IF NOT EXISTS registrant_full_legal_name text;

-- +goose Down
ALTER TABLE organization_documents DROP COLUMN IF EXISTS registrant_full_legal_name;

DROP INDEX IF EXISTS users_country_idx;

ALTER TABLE users DROP COLUMN IF EXISTS avatar_download_url;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_storage_key;
ALTER TABLE users DROP COLUMN IF EXISTS interests;
ALTER TABLE users DROP COLUMN IF EXISTS country;
ALTER TABLE users DROP COLUMN IF EXISTS date_of_birth;
ALTER TABLE users DROP COLUMN IF EXISTS gender;
