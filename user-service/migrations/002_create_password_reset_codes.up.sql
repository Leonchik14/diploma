-- +goose Up
CREATE TABLE IF NOT EXISTS password_reset_codes (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
	email TEXT NOT NULL,
	code_hash TEXT NOT NULL,
	salt TEXT NOT NULL,
	attempts INTEGER NOT NULL DEFAULT 0,
	max_attempts INTEGER NOT NULL DEFAULT 5,
	expires_at TIMESTAMPTZ NOT NULL,
	used_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_reset_codes_email_created ON password_reset_codes(email, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_password_reset_codes_email_expires ON password_reset_codes(email, expires_at);
CREATE INDEX IF NOT EXISTS idx_password_reset_codes_user_id ON password_reset_codes(user_id);
