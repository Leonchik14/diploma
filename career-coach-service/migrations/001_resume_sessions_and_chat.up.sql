-- +goose Up
CREATE TABLE IF NOT EXISTS resume_parse_sessions (
	session_id VARCHAR(36) PRIMARY KEY,
	user_id INTEGER NOT NULL,
	material_id VARCHAR(255),
	resume_profile_version BIGINT NOT NULL DEFAULT 1,
	questions_json JSONB,
	status VARCHAR(20) NOT NULL DEFAULT 'awaiting_user',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_resume_sessions_user_id ON resume_parse_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_resume_sessions_status ON resume_parse_sessions(status);
CREATE INDEX IF NOT EXISTS idx_resume_sessions_material_id ON resume_parse_sessions(material_id);

CREATE TABLE IF NOT EXISTS chat_conversations (
	conversation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id INTEGER NOT NULL,
	messages_json JSONB NOT NULL DEFAULT '[]'::jsonb,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_chat_conversations_user_id ON chat_conversations(user_id);
