package model

import "time"

// AskRequest represents the request to ask a question
type AskRequest struct {
	ConversationID string         `json:"conversation_id,omitempty"`
	Question       string         `json:"question"`
	ResumeProfile  *ResumeProfile `json:"resume_profile,omitempty"`
	ContextChunks  []ContextChunk `json:"context_chunks,omitempty"`
}

// AskResponse represents the response with LLM answer
type AskResponse struct {
	ConversationID string `json:"conversation_id"`
	Answer         string `json:"answer"`
}

// ResumeProfile represents user's resume information
type ResumeProfile struct {
	Role             string    `json:"role,omitempty"`
	Experience       string    `json:"experience,omitempty"`
	Skills           []string  `json:"skills,omitempty"`
	Location         string    `json:"location,omitempty"`
	SalaryExpectation *float64 `json:"salary_expectation,omitempty"`
}

// ContextChunk represents a retrieved document chunk
type ContextChunk struct {
	Source  string `json:"source"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ResumeParseRequest represents request to parse resume
type ResumeParseRequest struct {
	MaterialID string `json:"material_id"`
}

// ResumeParseResponse represents response from resume parsing
type ResumeParseResponse struct {
	SessionID string                 `json:"session_id"`
	Draft     *ResumeProfileDraft    `json:"draft"`
	Questions []Question             `json:"questions"`
	Status    string                 `json:"status"`
}

// ResumeAnswerRequest represents request to answer questions
type ResumeAnswerRequest struct {
	SessionID string          `json:"session_id"`
	Answers   []QuestionAnswer `json:"answers"`
}

// ResumeAnswerResponse represents response from answering questions
type ResumeAnswerResponse struct {
	SessionID string              `json:"session_id"`
	Draft     *ResumeProfileDraft `json:"draft"`
	Questions []Question          `json:"questions"`
	Status    string              `json:"status"`
}

// ResumeSessionRow is the DB row for a parse session (profile lives in user-service)
type ResumeSessionRow struct {
	SessionID            string     `json:"session_id"`
	UserID               uint       `json:"user_id"`
	MaterialID           string     `json:"material_id"`
	ResumeProfileVersion int64      `json:"resume_profile_version"`
	Questions            []Question `json:"questions"`
	Status               string     `json:"status"`
}

// ResumeSessionResponse represents session state returned to client (draft = current profile from user-service)
type ResumeSessionResponse struct {
	SessionID string              `json:"session_id"`
	Draft     *ResumeProfileDraft `json:"draft"`
	Questions []Question          `json:"questions"`
	Status    string              `json:"status"`
}

// ResumeProfileDraft represents draft resume profile
type ResumeProfileDraft struct {
	TargetRoles              []string                    `json:"target_roles"`
	ProfessionalRoleCandidates []ProfessionalRoleCandidate `json:"professional_role_candidates"`
	ExperienceLevel          *string                     `json:"experience_level"`
	Areas                    []Area                     `json:"areas"`
	SalaryMin                *float64                   `json:"salary_min"`
	Currency                 *string                    `json:"currency"`
	WorkFormat               []string                   `json:"work_format"`
	SkillsTop                []string                   `json:"skills_top"`
	Notes                    *string                    `json:"notes"`
	Confidence               map[string]float64         `json:"confidence"`
}

// ProfessionalRoleCandidate represents a candidate role
type ProfessionalRoleCandidate struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
}

// Area represents an area
type Area struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
}

// Question represents a question to user
type Question struct {
	ID      string   `json:"id"`
	Text    string   `json:"text"`
	Type    string   `json:"type"`
	Options []string `json:"options,omitempty"`
}

// QuestionAnswer represents user's answer
type QuestionAnswer struct {
	QuestionID string `json:"question_id"`
	Value      string `json:"value"`
}

// OpenRouterRequest represents the request to OpenRouter API
type OpenRouterRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse represents the response from OpenRouter API
type OpenRouterResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Error   *Error   `json:"error,omitempty"`
}

// Choice represents a choice in OpenRouter response
type Choice struct {
	Message Message `json:"message"`
}

// Error represents an error from OpenRouter API
type Error struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// ChatMessage represents a chat message in history
type ChatMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// CoachHistoryKind — тип записи в единой ленте «история с коучем».
type CoachHistoryKind int

const (
	CoachHistoryAskUser CoachHistoryKind = iota + 1
	CoachHistoryAskAssistant
	CoachHistoryReviewResume
	CoachHistoryPrepareVacancy
)

// CoachHistoryEntry — элемент объединённой истории (Ask + ReviewResume + PrepareForVacancy).
type CoachHistoryEntry struct {
	Kind           CoachHistoryKind
	ConversationID string
	Content        string
	ResumeScore    *float64
	VacancyID      string
	CreatedAt      time.Time
	StableOrder    int64 // больше = позже в потоке (при равном CreatedAt — выше в ленте «сначала новые»)
}
