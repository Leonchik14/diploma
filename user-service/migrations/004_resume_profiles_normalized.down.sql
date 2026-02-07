DROP TABLE IF EXISTS resume_profiles;

CREATE TABLE resume_profiles (
	user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
	profile_json JSONB NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_resume_profiles_user_id ON resume_profiles(user_id);
