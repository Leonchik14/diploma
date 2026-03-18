package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"career-coach-service/internal/database"
	"career-coach-service/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) CreateResumeSession(ctx context.Context, userID uint, materialID string, resumeProfileVersion int64, questions []model.Question) (string, error) {
	sessionID := uuid.New().String()
	questionsJSON, err := json.Marshal(questions)
	if err != nil {
		return "", err
	}
	_, err = database.DB.Exec(ctx,
		`INSERT INTO resume_parse_sessions (session_id, user_id, material_id, resume_profile_version, status, questions_json)
		 VALUES ($1, $2, $3, $4, 'awaiting_user', $5)`,
		sessionID, userID, materialID, resumeProfileVersion, questionsJSON)
	return sessionID, err
}

func (r *Repository) GetResumeSession(ctx context.Context, sessionID string, userID uint) (*model.ResumeSessionRow, error) {
	var questionsJSON []byte
	var status string
	var version int64
	var materialID string

	err := database.DB.QueryRow(ctx,
		`SELECT questions_json, status, resume_profile_version, material_id FROM resume_parse_sessions
		 WHERE session_id = $1 AND user_id = $2`,
		sessionID, userID).Scan(&questionsJSON, &status, &version, &materialID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}

	var questions []model.Question
	if len(questionsJSON) > 0 {
		_ = json.Unmarshal(questionsJSON, &questions)
	}

	return &model.ResumeSessionRow{
		SessionID:            sessionID,
		UserID:               userID,
		MaterialID:           materialID,
		ResumeProfileVersion: version,
		Questions:            questions,
		Status:               status,
	}, nil
}

func (r *Repository) UpdateResumeSession(ctx context.Context, sessionID string, userID uint, questions []model.Question, status string, version int64) error {
	questionsJSON, err := json.Marshal(questions)
	if err != nil {
		return err
	}
	_, err = database.DB.Exec(ctx,
		`UPDATE resume_parse_sessions SET questions_json = $1, status = $2, resume_profile_version = $3, updated_at = $4
		 WHERE session_id = $5 AND user_id = $6`,
		questionsJSON, status, version, time.Now(), sessionID, userID)
	return err
}

func (r *Repository) GetOrCreateConversation(ctx context.Context, conversationID string, userID uint) (string, error) {
	if conversationID != "" {
		var exists bool
		err := database.DB.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM chat_conversations WHERE conversation_id = $1 AND user_id = $2)`,
			conversationID, userID).Scan(&exists)

		if err != nil {
			return "", err
		}

		if exists {
			return conversationID, nil
		}
	}

	newID := uuid.New().String()
	_, err := database.DB.Exec(ctx,
		`INSERT INTO chat_conversations (conversation_id, user_id, messages_json)
		 VALUES ($1, $2, '[]'::jsonb)`,
		newID, userID)

	return newID, err
}

func (r *Repository) GetConversationMessages(ctx context.Context, conversationID string, userID uint, limit int) ([]model.ChatMessage, error) {
	var messagesJSON []byte

	err := database.DB.QueryRow(ctx,
		`SELECT messages_json FROM chat_conversations
		 WHERE conversation_id = $1 AND user_id = $2`,
		conversationID, userID).Scan(&messagesJSON)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("conversation not found")
		}
		return nil, err
	}

	var messages []model.ChatMessage
	if err := json.Unmarshal(messagesJSON, &messages); err != nil {
		return nil, err
	}

	if len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	return messages, nil
}

func (r *Repository) AddMessageToConversation(ctx context.Context, conversationID string, userID uint, role, content string, limit int) error {
	messages, err := r.GetConversationMessages(ctx, conversationID, userID, limit+1)
	if err != nil {
		return err
	}

	newMessage := model.ChatMessage{
		Role:      role,
		Content:   content,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	messages = append(messages, newMessage)

	if len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	_, err = database.DB.Exec(ctx,
		`UPDATE chat_conversations SET messages_json = $1, updated_at = $2
		 WHERE conversation_id = $3 AND user_id = $4`,
		messagesJSON, time.Now(), conversationID, userID)

	return err
}

// DeleteChatHistory removes chat rows for user. If conversationID is non-empty, only that conversation.
func (r *Repository) DeleteChatHistory(ctx context.Context, userID uint, conversationID string) (int64, error) {
	if conversationID != "" {
		tag, err := database.DB.Exec(ctx,
			`DELETE FROM chat_conversations WHERE user_id = $1 AND conversation_id = $2`,
			userID, conversationID)
		if err != nil {
			return 0, err
		}
		return tag.RowsAffected(), nil
	}
	tag, err := database.DB.Exec(ctx,
		`DELETE FROM chat_conversations WHERE user_id = $1`,
		userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *Repository) DeleteUserData(ctx context.Context, userID uint) error {
	_, err := database.DB.Exec(ctx,
		"DELETE FROM chat_conversations WHERE user_id = $1",
		userID)
	if err != nil {
		return err
	}

	_, err = database.DB.Exec(ctx,
		"DELETE FROM resume_parse_sessions WHERE user_id = $1",
		userID)
	if err != nil {
		return err
	}
	_, err = database.DB.Exec(ctx,
		"DELETE FROM coach_interaction_history WHERE user_id = $1",
		userID)
	return err
}

// UserConversationRow — диалог и сырой JSON сообщений.
type UserConversationRow struct {
	ConversationID string
	MessagesJSON   []byte
	UpdatedAt      time.Time
}

func (r *Repository) ListUserConversations(ctx context.Context, userID uint) ([]UserConversationRow, error) {
	rows, err := database.DB.Query(ctx,
		`SELECT conversation_id::text, messages_json, updated_at
		 FROM chat_conversations WHERE user_id = $1 ORDER BY updated_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UserConversationRow
	for rows.Next() {
		var row UserConversationRow
		if err := rows.Scan(&row.ConversationID, &row.MessagesJSON, &row.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) InsertCoachInteraction(ctx context.Context, userID uint, eventType, body string, meta map[string]any) error {
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	_, err = database.DB.Exec(ctx,
		`INSERT INTO coach_interaction_history (user_id, event_type, body, meta) VALUES ($1, $2, $3, $4::jsonb)`,
		userID, eventType, body, metaJSON)
	return err
}

func (r *Repository) ListCoachInteractions(ctx context.Context, userID uint) ([]CoachInteractionRow, error) {
	rows, err := database.DB.Query(ctx,
		`SELECT id, event_type, body, meta, created_at FROM coach_interaction_history
		 WHERE user_id = $1 ORDER BY created_at ASC, id ASC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CoachInteractionRow
	for rows.Next() {
		var row CoachInteractionRow
		if err := rows.Scan(&row.ID, &row.EventType, &row.Body, &row.MetaJSON, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) DeleteCoachInteractionsByUser(ctx context.Context, userID uint) error {
	_, err := database.DB.Exec(ctx, `DELETE FROM coach_interaction_history WHERE user_id = $1`, userID)
	return err
}

// CoachInteractionRow — запись ReviewResume / PrepareForVacancy.
type CoachInteractionRow struct {
	ID        int64
	EventType string
	Body      string
	MetaJSON  []byte
	CreatedAt time.Time
}
