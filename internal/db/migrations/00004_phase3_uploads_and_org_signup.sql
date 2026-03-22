-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'upload_status') THEN
		CREATE TYPE upload_status AS ENUM ('pending', 'completed');
	END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS uploads (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	storage_key text NOT NULL UNIQUE,
	file_name text NOT NULL,
	file_size bigint NOT NULL,
	mime_type text NOT NULL,
	status upload_status NOT NULL DEFAULT 'pending',
	created_at timestamptz NOT NULL DEFAULT NOW(),
	completed_at timestamptz
);

-- +goose Down
DROP TABLE IF EXISTS uploads;
-- +goose StatementBegin
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'upload_status') THEN
		DROP TYPE upload_status;
	END IF;
END
$$;
-- +goose StatementEnd
