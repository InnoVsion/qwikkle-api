-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

-- +goose StatementBegin
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
		CREATE TYPE user_role AS ENUM ('user', 'admin', 'editor');
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'account_status') THEN
		CREATE TYPE account_status AS ENUM ('active', 'suspended', 'deactivated');
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'verification_status') THEN
		CREATE TYPE verification_status AS ENUM ('pending', 'approved', 'rejected');
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'document_type') THEN
		CREATE TYPE document_type AS ENUM (
			'registration_certificate',
			'tax_id',
			'proof_of_address',
			'id_document',
			'other'
		);
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'document_status') THEN
		CREATE TYPE document_status AS ENUM ('pending', 'approved', 'rejected');
	END IF;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
	NEW.updated_at = NOW();
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS users (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	qk_id citext NOT NULL UNIQUE,
	email citext UNIQUE,
	password_hash text NOT NULL,
	role user_role NOT NULL DEFAULT 'user',
	status account_status NOT NULL DEFAULT 'active',
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW(),
	last_login_at timestamptz,
	CONSTRAINT users_qk_id_format CHECK (qk_id ~* '^[a-z0-9](?:[a-z0-9._-]*[a-z0-9])?\\.qk$')
);

DROP TRIGGER IF EXISTS users_set_updated_at ON users;
CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS organizations (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	owner_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
	name text NOT NULL,
	email citext,
	phone text,
	website text,
	social_handle text,
	status account_status NOT NULL DEFAULT 'active',
	verification_status verification_status NOT NULL DEFAULT 'pending',
	created_at timestamptz NOT NULL DEFAULT NOW(),
	updated_at timestamptz NOT NULL DEFAULT NOW()
);

DROP TRIGGER IF EXISTS organizations_set_updated_at ON organizations;
CREATE TRIGGER organizations_set_updated_at
BEFORE UPDATE ON organizations
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS organization_documents (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
	uploaded_by_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
	type document_type NOT NULL,
	status document_status NOT NULL DEFAULT 'pending',
	file_name text NOT NULL,
	file_size bigint NOT NULL,
	mime_type text NOT NULL,
	storage_key text NOT NULL,
	rejection_reason text,
	uploaded_at timestamptz NOT NULL DEFAULT NOW(),
	reviewed_at timestamptz,
	reviewed_by_user_id uuid REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS organization_documents_org_id_idx
ON organization_documents (organization_id);

CREATE INDEX IF NOT EXISTS organization_documents_status_idx
ON organization_documents (status);

CREATE TABLE IF NOT EXISTS sessions (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	refresh_token_hash text NOT NULL,
	created_at timestamptz NOT NULL DEFAULT NOW(),
	expires_at timestamptz NOT NULL,
	revoked_at timestamptz,
	user_agent text,
	ip inet
);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx
ON sessions (user_id);

CREATE INDEX IF NOT EXISTS sessions_expires_at_idx
ON sessions (expires_at);

-- +goose Down
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS organization_documents;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS set_updated_at();

-- +goose StatementBegin
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'document_status') THEN
		DROP TYPE document_status;
	END IF;
	IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'document_type') THEN
		DROP TYPE document_type;
	END IF;
	IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'verification_status') THEN
		DROP TYPE verification_status;
	END IF;
	IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'account_status') THEN
		DROP TYPE account_status;
	END IF;
	IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
		DROP TYPE user_role;
	END IF;
END
$$;
-- +goose StatementEnd
