-- +goose Up
DROP TABLE IF EXISTS resume_profiles;

CREATE TABLE resume_profiles (
	user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
	status TEXT NOT NULL CHECK (status IN ('DRAFT', 'CONFIRMED')) DEFAULT 'DRAFT',
	source_material_id TEXT,
	target_roles TEXT[] NOT NULL DEFAULT '{}',
	experience_level TEXT,
	area_ids TEXT[] NOT NULL DEFAULT '{}',
	salary_min INTEGER,
	currency TEXT,
	work_format TEXT[],
	skills_top TEXT[] NOT NULL DEFAULT '{}',
	education_level TEXT,
	notes TEXT,
	confidence JSONB NOT NULL DEFAULT '{}',
	confirmed_fields JSONB NOT NULL DEFAULT '{}',
	version BIGINT NOT NULL DEFAULT 1,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_resume_profiles_user_id ON resume_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_resume_profiles_status ON resume_profiles(status);
