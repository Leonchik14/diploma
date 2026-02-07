package service

import (
	"context"
	"fmt"
	"strings"

	"career-coach-service/internal/llm"
	"career-coach-service/internal/model"
	"career-coach-service/internal/repository"
)

type CoachService struct {
	llmClient        *llm.Client
	model            string
	repo             *repository.Repository
	chatHistoryLimit int
}

func NewCoachService(llmClient *llm.Client, repo *repository.Repository, model string, chatHistoryLimit int) *CoachService {
	return &CoachService{
		llmClient:        llmClient,
		model:            model,
		repo:             repo,
		chatHistoryLimit: chatHistoryLimit,
	}
}

func (s *CoachService) Ask(ctx context.Context, userID uint, req *model.AskRequest) (*model.AskResponse, error) {
	conversationID, err := s.repo.GetOrCreateConversation(ctx, req.ConversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create conversation: %w", err)
	}

	history, err := s.repo.GetConversationMessages(ctx, conversationID, userID, s.chatHistoryLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}

	systemPrompt := s.buildSystemPrompt()
	userMessage := s.buildUserMessage(req, history)

	messages := []model.Message{
		{Role: "system", Content: systemPrompt},
	}

	for _, msg := range history {
		messages = append(messages, model.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	messages = append(messages, model.Message{
		Role:    "user",
		Content: userMessage,
	})

	answer, err := s.llmClient.ChatCompletion(ctx, s.model, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response: %w", err)
	}

	if err := s.repo.AddMessageToConversation(ctx, conversationID, userID, "user", req.Question, s.chatHistoryLimit); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	if err := s.repo.AddMessageToConversation(ctx, conversationID, userID, "assistant", answer, s.chatHistoryLimit); err != nil {
		return nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	return &model.AskResponse{
		ConversationID: conversationID,
		Answer:         answer,
	}, nil
}

func (s *CoachService) DeleteUserData(ctx context.Context, userID uint) error {
	return s.repo.DeleteUserData(ctx, userID)
}

func (s *CoachService) buildSystemPrompt() string {
	return `Ты профессиональный карьерный коуч и эксперт по подготовке к собеседованиям. Твоя роль - помогать пользователям готовиться к собеседованиям и развивать карьеру.

ВАЖНЫЕ ПРАВИЛА:
1. НИКОГДА не выдумывай информацию. Используй только информацию, предоставленную в вопросе пользователя и контексте.
2. Если у тебя недостаточно информации для ответа, задавай уточняющие вопросы вместо того, чтобы угадывать.
3. Всегда основывай свои советы на предоставленных фрагментах контекста и профиле резюме пользователя.
4. Будь профессиональным, поддерживающим и конструктивным в своих ответах.
5. Фокусируйся на практических, применимых советах.
6. Если в контексте нет релевантной информации, явно укажи, что тебе нужна дополнительная информация.

ВАЖНО: Всегда отвечай на русском языке.

Помни: Ты здесь, чтобы помочь пользователям добиться успеха в их карьерном пути.`
}

func (s *CoachService) buildUserMessage(req *model.AskRequest, history []model.ChatMessage) string {
	var parts []string

	if req.ResumeProfile != nil {
		parts = append(parts, "--- Resume Profile ---")
		if req.ResumeProfile.Role != "" {
			parts = append(parts, fmt.Sprintf("Role: %s", req.ResumeProfile.Role))
		}
		if req.ResumeProfile.Experience != "" {
			parts = append(parts, fmt.Sprintf("Experience: %s", req.ResumeProfile.Experience))
		}
		if len(req.ResumeProfile.Skills) > 0 {
			parts = append(parts, fmt.Sprintf("Skills: %s", strings.Join(req.ResumeProfile.Skills, ", ")))
		}
		if req.ResumeProfile.Location != "" {
			parts = append(parts, fmt.Sprintf("Location: %s", req.ResumeProfile.Location))
		}
		if req.ResumeProfile.SalaryExpectation != nil {
			parts = append(parts, fmt.Sprintf("Salary Expectation: %.0f", *req.ResumeProfile.SalaryExpectation))
		}
		parts = append(parts, "")
	}

	contextChunks := req.ContextChunks
	if len(contextChunks) > 5 {
		contextChunks = contextChunks[:5]
	}

	if len(contextChunks) > 0 {
		parts = append(parts, "--- Relevant Context ---")
		for i, chunk := range contextChunks {
			parts = append(parts, fmt.Sprintf("\n[Context %d - Source: %s]", i+1, chunk.Source))
			if chunk.Title != "" {
				parts = append(parts, fmt.Sprintf("Title: %s", chunk.Title))
			}
			parts = append(parts, fmt.Sprintf("Content: %s", chunk.Content))
		}
		parts = append(parts, "")
	}

	parts = append(parts, fmt.Sprintf("Question: %s", req.Question))

	return strings.Join(parts, "\n")
}
