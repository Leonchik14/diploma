CREATE TABLE IF NOT EXISTS calendar_events (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id INTEGER NOT NULL,
	title TEXT NOT NULL,
	description TEXT,
	event_type SMALLINT NOT NULL,
	start_time TIMESTAMPTZ NOT NULL,
	end_time TIMESTAMPTZ NOT NULL,
	timezone TEXT,
	location TEXT,
	related_vacancy_id TEXT,
	reminder_minutes INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_calendar_events_user_start ON calendar_events(user_id, start_time);
CREATE INDEX IF NOT EXISTS idx_calendar_events_user_end ON calendar_events(user_id, end_time);
CREATE INDEX IF NOT EXISTS idx_calendar_events_user_id ON calendar_events(user_id, id);
