-- +goose Up
CREATE TABLE IF NOT EXISTS coach_interaction_history (
	id BIGSERIAL PRIMARY KEY,
	user_id INTEGER NOT NULL,
	event_type VARCHAR(32) NOT NULL,
	body TEXT NOT NULL,
	meta JSONB NOT NULL DEFAULT '{}',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_coach_interaction_user_created ON coach_interaction_history (user_id, created_at DESC);
