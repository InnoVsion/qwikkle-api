-- +goose Up
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_qk_id_format;
ALTER TABLE users
	ADD CONSTRAINT users_qk_id_format
	CHECK (qk_id::text ~* '^[a-z0-9](?:[a-z0-9._-]*[a-z0-9])?\.qk$');

-- +goose Down
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_qk_id_format;
ALTER TABLE users
	ADD CONSTRAINT users_qk_id_format
	CHECK (qk_id ~* '^[a-z0-9](?:[a-z0-9._-]*[a-z0-9])?\.qk$');
