-- +goose Up
ALTER TABLE resume_parse_sessions ADD COLUMN IF NOT EXISTS resume_profile_version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE resume_parse_sessions DROP COLUMN IF EXISTS draft_json;
