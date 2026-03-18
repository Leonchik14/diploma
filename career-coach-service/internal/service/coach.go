package service

import (
	"context"
	"fmt"
	"strings"

	"career-coach-service/internal/client"
	"career-coach-service/internal/llm"
	"career-coach-service/internal/model"
	"career-coach-service/internal/repository"

	pbjobs "proto/jobs"
	pbuser "proto/user"
)

type CoachService struct {
	llmClient        *llm.Client
	model            string
	repo             *repository.Repository
	chatHistoryLimit int
	jobsClient       *client.JobsClient
	userClient       *client.UserClient
}

func NewCoachService(llmClient *llm.Client, repo *repository.Repository, model string, chatHistoryLimit int, jobsClient *client.JobsClient, userClient *client.UserClient) *CoachService {
	return &CoachService{
		llmClient:        llmClient,
		model:            model,
		repo:             repo,
		chatHistoryLimit: chatHistoryLimit,
		jobsClient:       jobsClient,
		userClient:       userClient,
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

func (s *CoachService) ClearChatHistory(ctx context.Context, userID uint, conversationID string) (deleted int64, err error) {
	deleted, err = s.repo.DeleteChatHistory(ctx, userID, conversationID)
	if err != nil {
		return 0, fmt.Errorf("clear chat history: %w", err)
	}
	return deleted, nil
}

func (s *CoachService) DeleteUserData(ctx context.Context, userID uint) error {
	return s.repo.DeleteUserData(ctx, userID)
}

func (s *CoachService) PrepareForVacancy(ctx context.Context, userID uint, vacancyID string) (string, error) {
	if s.jobsClient == nil {
		return "", fmt.Errorf("jobs client not configured")
	}

	vacancy, err := s.jobsClient.GetVacancy(ctx, vacancyID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch vacancy: %w", err)
	}

	vacancyText := formatVacancyForLLM(vacancy)
	systemPrompt := `Ты профессиональный карьерный коуч. На основе описания вакансии с hh.ru дай конкретные рекомендации по подготовке к собеседованию: что изучить, какие вопросы могут задать, на что обратить внимание. Будь практичным и конкретным. Отвечай на русском языке.`
	userMessage := fmt.Sprintf("Вакансия:\n%s\n\nДай рекомендации по подготовке к собеседованию на эту вакансию.", vacancyText)

	messages := []model.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	recommendations, err := s.llmClient.ChatCompletion(ctx, s.model, messages)
	if err != nil {
		return "", fmt.Errorf("failed to get LLM response: %w", err)
	}
	return recommendations, nil
}

func (s *CoachService) ReviewResume(ctx context.Context, userID uint) (score float64, recommendations string, err error) {
	if s.userClient == nil {
		return 0, "", fmt.Errorf("user client not configured")
	}

	resp, err := s.userClient.GetResumeProfileInternal(ctx, userID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get resume profile: %w", err)
	}
	if resp.Profile == nil {
		return 0, "", fmt.Errorf("resume profile not found")
	}

	profileText := formatResumeProfileForLLM(resp.Profile)
	systemPrompt := `Ты профессиональный HR-эксперт и карьерный консультант. Проанализируй резюме пользователя и:
1. Поставь оценку от 1 до 10 (целое число или с одним знаком после запятой).
2. Дай конкретные рекомендации по улучшению резюме: что добавить, что улучшить, какие формулировки изменить.

Формат ответа:
ОЦЕНКА: X/10

РЕКОМЕНДАЦИИ:
- пункт 1
- пункт 2
...

Отвечай на русском языке.`
	userMessage := fmt.Sprintf("Резюме пользователя:\n%s\n\nПроанализируй и дай оценку с рекомендациями.", profileText)

	messages := []model.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	answer, err := s.llmClient.ChatCompletion(ctx, s.model, messages)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get LLM response: %w", err)
	}

	score = parseScoreFromLLMResponse(answer)
	return score, answer, nil
}

func formatVacancyForLLM(v *pbjobs.Vacancy) string {
	if v == nil {
		return ""
	}
	var parts []string
	parts = append(parts, fmt.Sprintf("Название: %s", v.Name))
	parts = append(parts, fmt.Sprintf("Описание: %s", v.Description))
	if v.Employer != nil && v.Employer.Name != "" {
		parts = append(parts, fmt.Sprintf("Работодатель: %s", v.Employer.Name))
	}
	if v.Area != nil && v.Area.Name != "" {
		parts = append(parts, fmt.Sprintf("Регион: %s", v.Area.Name))
	}
	if v.Salary != nil {
		var s string
		if v.Salary.From != nil && v.Salary.To != nil {
			s = fmt.Sprintf("%d - %d %s", *v.Salary.From, *v.Salary.To, v.Salary.Currency)
		} else if v.Salary.From != nil {
			s = fmt.Sprintf("от %d %s", *v.Salary.From, v.Salary.Currency)
		} else if v.Salary.To != nil {
			s = fmt.Sprintf("до %d %s", *v.Salary.To, v.Salary.Currency)
		} else {
			s = v.Salary.Currency
		}
		if s != "" {
			parts = append(parts, fmt.Sprintf("Зарплата: %s", s))
		}
	}
	if v.Experience != nil && *v.Experience != "" {
		parts = append(parts, fmt.Sprintf("Опыт: %s", *v.Experience))
	}
	return strings.Join(parts, "\n")
}

func formatResumeProfileForLLM(p *pbuser.ResumeProfile) string {
	if p == nil {
		return ""
	}
	var parts []string
	if len(p.TargetRoles) > 0 {
		parts = append(parts, fmt.Sprintf("Целевые роли: %s", strings.Join(p.TargetRoles, ", ")))
	}
	if p.ExperienceLevel != nil {
		parts = append(parts, fmt.Sprintf("Уровень опыта: %s", *p.ExperienceLevel))
	}
	if len(p.Areas) > 0 {
		areaNames := make([]string, len(p.Areas))
		for i, a := range p.Areas {
			areaNames[i] = a.Name
		}
		parts = append(parts, fmt.Sprintf("Регионы: %s", strings.Join(areaNames, ", ")))
	}
	if p.SalaryMin != nil {
		currency := "RUR"
		if p.Currency != nil {
			currency = *p.Currency
		}
		parts = append(parts, fmt.Sprintf("Желаемая ЗП: %.0f %s", *p.SalaryMin, currency))
	}
	if len(p.WorkFormat) > 0 {
		parts = append(parts, fmt.Sprintf("Формат работы: %s", strings.Join(p.WorkFormat, ", ")))
	}
	if len(p.SkillsTop) > 0 {
		parts = append(parts, fmt.Sprintf("Навыки: %s", strings.Join(p.SkillsTop, ", ")))
	}
	if p.EducationLevel != nil {
		parts = append(parts, fmt.Sprintf("Образование: %s", *p.EducationLevel))
	}
	if p.Notes != nil {
		parts = append(parts, fmt.Sprintf("Заметки: %s", *p.Notes))
	}
	return strings.Join(parts, "\n")
}

func parseScoreFromLLMResponse(answer string) float64 {
	// Simple heuristic: look for "X/10" or "ОЦЕНКА: X"
	lines := strings.Split(answer, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(strings.ToUpper(line))
		if strings.Contains(line, "/10") {
			var score float64
			if _, err := fmt.Sscanf(line, "%f/10", &score); err == nil && score >= 0 && score <= 10 {
				return score
			}
		}
	}
	return 0
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
