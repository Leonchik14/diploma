package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"career-coach-service/internal/llm"
	"career-coach-service/internal/model"
)

type Parser struct {
	llmClient    *llm.Client
	parseModel  string
	maxChars    int
}

func NewParser(llmClient *llm.Client, parseModel string, maxChars int) *Parser {
	return &Parser{
		llmClient:   llmClient,
		parseModel:  parseModel,
		maxChars:    maxChars,
	}
}

func (p *Parser) ParseResume(ctx context.Context, text string) (*model.ResumeProfileDraft, []model.Question, error) {
	cleanedText := p.cleanText(text)
	if len(cleanedText) > p.maxChars {
		cleanedText = cleanedText[:p.maxChars]
	}

	systemPrompt := p.buildParseSystemPrompt()
	userPrompt := fmt.Sprintf("Parse the following resume text and extract structured information:\n\n%s", cleanedText)

	messages := []model.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := p.llmClient.ChatCompletion(ctx, p.parseModel, messages)
	if err != nil {
		return nil, nil, err
	}

	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, nil, fmt.Errorf("invalid JSON response from LLM")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var result struct {
		Draft     model.ResumeProfileDraft `json:"draft"`
		Questions []model.Question         `json:"questions"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return &result.Draft, result.Questions, nil
}

func (p *Parser) cleanText(text string) string {
	text = p.removePII(text)
	text = strings.TrimSpace(text)
	return text
}

func (p *Parser) removePII(text string) string {
	emailRegex := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
	text = emailRegex.ReplaceAllString(text, "[EMAIL]")

	phoneRegex := regexp.MustCompile(`\b\d{1,3}[-.\s]?\(?\d{1,4}\)?[-.\s]?\d{1,4}[-.\s]?\d{1,9}\b`)
	text = phoneRegex.ReplaceAllString(text, "[PHONE]")

	return text
}

func (p *Parser) buildParseSystemPrompt() string {
	return `You are a resume parsing assistant. Extract structured information from resume text.

CRITICAL RULES:
1. Return ONLY valid JSON, no markdown, no code blocks.
2. Fill ResumeProfileDraft with extracted information.
3. Set confidence (0.0-1.0) for each important field.
4. If uncertain, set field to null and confidence < 0.6.
5. Extract 5-10 top skills into skills_top.
6. ALWAYS extract target_roles: infer 1-3 desired/likely job roles from the resume (current job title, last position, "looking for" section, or from experience and skills). Put the most relevant role first. Examples: "Backend Developer", "Go Developer", "Software Engineer". Do not leave target_roles empty if the resume contains any job title or role description.
7. Optionally fill professional_role_candidates with alternative roles and confidence scores.
8. Generate 3-4 clarifying questions ONLY about fields with low confidence: target_roles, experience_level, areas, salary_min, work_format.

Response format:
{
  "draft": {
    "target_roles": ["Primary Role", "Another Role"],
    "professional_role_candidates": [{"id": "...", "name": "...", "confidence": 0.0}],
    "experience_level": null,
    "areas": [{"id": "...", "name": "...", "confidence": 0.0}],
    "salary_min": null,
    "currency": null,
    "work_format": [],
    "skills_top": [],
    "notes": null,
    "confidence": {"target_roles": 0.9, ...}
  },
  "questions": [
    {"id": "target_roles", "text": "...", "type": "single_choice|free_text", "options": []}
  ]
}`
}

func (p *Parser) ApplyAnswers(draft *model.ResumeProfileDraft, questions []model.Question, answers []model.QuestionAnswer) (*model.ResumeProfileDraft, []model.Question, string) {
	answerMap := make(map[string]string)
	for _, ans := range answers {
		answerMap[ans.QuestionID] = ans.Value
	}

	updatedQuestions := []model.Question{}
	allConfirmed := true

	for _, q := range questions {
		if answer, ok := answerMap[q.ID]; ok {
			switch q.ID {
			case "target_roles":
				if answer != "" {
					draft.TargetRoles = []string{answer}
					draft.Confidence["target_roles"] = 1.0
				}
			case "experience_level":
				if answer != "" {
					draft.ExperienceLevel = &answer
					draft.Confidence["experience_level"] = 1.0
				}
			case "salary_min":
				var salary float64
				if _, err := fmt.Sscanf(answer, "%f", &salary); err == nil {
					draft.SalaryMin = &salary
					draft.Confidence["salary_min"] = 1.0
				}
			case "work_format":
				if answer != "" {
					draft.WorkFormat = []string{answer}
					draft.Confidence["work_format"] = 1.0
				}
			}
		} else {
			if draft.Confidence[q.ID] < 0.6 {
				updatedQuestions = append(updatedQuestions, q)
				allConfirmed = false
			}
		}
	}

	status := "awaiting_user"
	if allConfirmed && len(updatedQuestions) == 0 {
		status = "completed"
	}

	return draft, updatedQuestions, status
}
