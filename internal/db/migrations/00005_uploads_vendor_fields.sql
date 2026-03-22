-- +goose Up
ALTER TABLE uploads
	ADD COLUMN IF NOT EXISTS provider text NOT NULL DEFAULT 's3',
	ADD COLUMN IF NOT EXISTS download_url text;

CREATE INDEX IF NOT EXISTS uploads_provider_idx ON uploads (provider);

-- +goose Down
DROP INDEX IF EXISTS uploads_provider_idx;
ALTER TABLE uploads DROP COLUMN IF EXISTS download_url;
ALTER TABLE uploads DROP COLUMN IF EXISTS provider;
