package database

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func Connect(databaseURL string) error {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return err
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 5 * time.Minute
	config.MaxConnIdleTime = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return err
	}

	DB = pool
	log.Println("Connected to PostgreSQL database")

	return AutoMigrate()
}

func AutoMigrate() error {
	ctx := context.Background()

	eventsTable := `
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
	`

	if _, err := DB.Exec(ctx, eventsTable); err != nil {
		return err
	}

	log.Println("Database migrations completed")
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
