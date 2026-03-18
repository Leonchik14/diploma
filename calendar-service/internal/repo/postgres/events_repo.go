package postgres

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"calendar-service/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventsRepo struct {
	db *pgxpool.Pool
}

func NewEventsRepo(db *pgxpool.Pool) *EventsRepo {
	return &EventsRepo{db: db}
}

func (r *EventsRepo) Create(ctx context.Context, event *model.Event) error {
	event.ID = uuid.New().String()
	now := time.Now()
	event.CreatedAt = now
	event.UpdatedAt = now

	_, err := r.db.Exec(ctx,
		`INSERT INTO calendar_events 
		 (id, user_id, title, description, event_type, start_time, end_time, timezone, location, related_vacancy_id, reminder_minutes, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		event.ID, event.UserID, event.Title, event.Description, event.EventType,
		event.StartTime, event.EndTime, event.Timezone, event.Location,
		event.RelatedVacancyID, event.ReminderMinutes, event.CreatedAt, event.UpdatedAt)
	return err
}

func (r *EventsRepo) GetByID(ctx context.Context, userID uint, eventID string) (*model.Event, error) {
	var e model.Event
	var description, timezone, location, relatedVacancyID sql.NullString

	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, title, description, event_type, start_time, end_time, 
		 timezone, location, related_vacancy_id, reminder_minutes, created_at, updated_at
		 FROM calendar_events WHERE id = $1 AND user_id = $2`,
		eventID, userID).Scan(
		&e.ID, &e.UserID, &e.Title, &description, &e.EventType,
		&e.StartTime, &e.EndTime, &timezone, &location, &relatedVacancyID,
		&e.ReminderMinutes, &e.CreatedAt, &e.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("event not found")
		}
		return nil, err
	}

	if description.Valid {
		e.Description = &description.String
	}
	if timezone.Valid {
		e.Timezone = &timezone.String
	}
	if location.Valid {
		e.Location = &location.String
	}
	if relatedVacancyID.Valid {
		e.RelatedVacancyID = &relatedVacancyID.String
	}

	return &e, nil
}

func (r *EventsRepo) Update(ctx context.Context, userID uint, eventID string, updateFn func(*model.Event)) error {
	event, err := r.GetByID(ctx, userID, eventID)
	if err != nil {
		return err
	}

	updateFn(event)
	event.UpdatedAt = time.Now()

	_, err = r.db.Exec(ctx,
		`UPDATE calendar_events SET 
		 title = $1, description = $2, event_type = $3, start_time = $4, end_time = $5,
		 timezone = $6, location = $7, related_vacancy_id = $8, reminder_minutes = $9, updated_at = $10
		 WHERE id = $11 AND user_id = $12`,
		event.Title, event.Description, event.EventType, event.StartTime, event.EndTime,
		event.Timezone, event.Location, event.RelatedVacancyID, event.ReminderMinutes,
		event.UpdatedAt, eventID, userID)
	return err
}

func (r *EventsRepo) Delete(ctx context.Context, userID uint, eventID string) error {
	result, err := r.db.Exec(ctx,
		"DELETE FROM calendar_events WHERE id = $1 AND user_id = $2", eventID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("event not found")
	}
	return nil
}

func (r *EventsRepo) List(ctx context.Context, userID uint, fromTime, toTime time.Time, pageSize int32, pageToken string, sortAsc bool) ([]*model.Event, string, error) {
	var cursor *model.PageToken
	if pageToken != "" {
		var err error
		cursor, err = decodePageToken(pageToken)
		if err != nil {
			return nil, "", fmt.Errorf("invalid page token")
		}
	}

	query := `SELECT id, user_id, title, description, event_type, start_time, end_time, 
			  timezone, location, related_vacancy_id, reminder_minutes, created_at, updated_at
			  FROM calendar_events WHERE user_id = $1 AND start_time >= $2 AND end_time < $3`

	args := []interface{}{userID, fromTime, toTime}
	argIdx := 4

	if cursor != nil {
		if sortAsc {
			query += fmt.Sprintf(" AND (start_time > $%d OR (start_time = $%d AND id > $%d))", argIdx, argIdx, argIdx+1)
		} else {
			query += fmt.Sprintf(" AND (start_time < $%d OR (start_time = $%d AND id < $%d))", argIdx, argIdx, argIdx+1)
		}
		args = append(args, cursor.StartTime, cursor.ID)
		argIdx += 2
	}

	if sortAsc {
		query += " ORDER BY start_time ASC, id ASC"
	} else {
		query += " ORDER BY start_time DESC, id DESC"
	}

	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, pageSize+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	events := make([]*model.Event, 0)
	for rows.Next() {
		var e model.Event
		var description, timezone, location, relatedVacancyID sql.NullString

		err := rows.Scan(
			&e.ID, &e.UserID, &e.Title, &description, &e.EventType,
			&e.StartTime, &e.EndTime, &timezone, &location, &relatedVacancyID,
			&e.ReminderMinutes, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, "", err
		}

		if description.Valid {
			e.Description = &description.String
		}
		if timezone.Valid {
			e.Timezone = &timezone.String
		}
		if location.Valid {
			e.Location = &location.String
		}
		if relatedVacancyID.Valid {
			e.RelatedVacancyID = &relatedVacancyID.String
		}

		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	var nextToken string
	if len(events) > int(pageSize) {
		last := events[pageSize]
		nextToken = encodePageToken(&model.PageToken{
			StartTime: last.StartTime,
			ID:        last.ID,
		})
		events = events[:pageSize]
	}

	return events, nextToken, nil
}

func (r *EventsRepo) ListUpcoming(ctx context.Context, userID uint, fromTime time.Time, limit int32) ([]*model.Event, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, title, description, event_type, start_time, end_time, 
		 timezone, location, related_vacancy_id, reminder_minutes, created_at, updated_at
		 FROM calendar_events WHERE user_id = $1 AND start_time >= $2
		 ORDER BY start_time ASC, id ASC LIMIT $3`,
		userID, fromTime, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*model.Event, 0)
	for rows.Next() {
		var e model.Event
		var description, timezone, location, relatedVacancyID sql.NullString

		err := rows.Scan(
			&e.ID, &e.UserID, &e.Title, &description, &e.EventType,
			&e.StartTime, &e.EndTime, &timezone, &location, &relatedVacancyID,
			&e.ReminderMinutes, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			e.Description = &description.String
		}
		if timezone.Valid {
			e.Timezone = &timezone.String
		}
		if location.Valid {
			e.Location = &location.String
		}
		if relatedVacancyID.Valid {
			e.RelatedVacancyID = &relatedVacancyID.String
		}

		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func encodePageToken(token *model.PageToken) string {
	data, _ := json.Marshal(token)
	return base64.URLEncoding.EncodeToString(data)
}

func decodePageToken(tokenStr string) (*model.PageToken, error) {
	data, err := base64.URLEncoding.DecodeString(tokenStr)
	if err != nil {
		return nil, err
	}
	var token model.PageToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// Счётчики по событиям типа INTERVIEW (1): upcoming — ещё не начались, completed — уже закончились, total — все.
func (r *EventsRepo) CountInterviews(ctx context.Context, userID uint) (upcoming, completed, total int32, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT 
			COUNT(*) FILTER (WHERE start_time > NOW()),
			COUNT(*) FILTER (WHERE end_time < NOW()),
			COUNT(*)
		 FROM calendar_events 
		 WHERE user_id = $1 AND event_type = 1`,
		userID).Scan(&upcoming, &completed, &total)
	return
}

func (r *EventsRepo) DeleteUserData(ctx context.Context, userID uint) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM calendar_events WHERE user_id = $1",
		userID)
	return err
}
