-- +goose Up
ALTER TABLE organizations
	ADD COLUMN IF NOT EXISTS logo_url text;

ALTER TABLE users
	ADD COLUMN IF NOT EXISTS first_name text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS last_name text NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS phone text,
	ADD COLUMN IF NOT EXISTS avatar_url text,
	ADD COLUMN IF NOT EXISTS last_active_at timestamptz,
	ADD COLUMN IF NOT EXISTS organization_id uuid REFERENCES organizations(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS organization_members (
	organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	role text NOT NULL DEFAULT 'member',
	created_at timestamptz NOT NULL DEFAULT NOW(),
	PRIMARY KEY (organization_id, user_id)
);

ALTER TABLE organization_documents
	ADD COLUMN IF NOT EXISTS download_url text;

-- +goose Down
ALTER TABLE organization_documents DROP COLUMN IF EXISTS download_url;
DROP TABLE IF EXISTS organization_members;
ALTER TABLE users DROP COLUMN IF EXISTS organization_id;
ALTER TABLE users DROP COLUMN IF EXISTS last_active_at;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE users DROP COLUMN IF EXISTS phone;
ALTER TABLE users DROP COLUMN IF EXISTS last_name;
ALTER TABLE users DROP COLUMN IF EXISTS first_name;
ALTER TABLE organizations DROP COLUMN IF EXISTS logo_url;
