package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"career-coach-service/internal/llm"
	"career-coach-service/internal/model"
)

type Parser struct {
	llmClient    *llm.Client
	parseModel  string
	maxChars    int
	areasByID   map[string]string
	areaIDByName map[string]string
}

var defaultAreaQuestionOptions = []string{
	"Москва",
	"Санкт-Петербург",
	"Казань",
	"Новосибирск",
	"Екатеринбург",
	"Нижний Новгород",
	"Самара",
	"Краснодар",
	"Ростов-на-Дону",
	"Россия",
}

var workFormatQuestionOptions = []string{"Удаленно", "Гибрид", "Офис"}
var experienceQuestionOptions = []string{"Нет опыта", "1-3 года", "3-6 лет", "6+ лет"}
var salaryQuestionPresets = []string{"50000", "100000", "150000", "200000"}

func NewParser(llmClient *llm.Client, parseModel string, maxChars int) *Parser {
	areasByID, areaIDByName := loadHHAreasCatalog()
	return &Parser{
		llmClient:   llmClient,
		parseModel:  parseModel,
		maxChars:    maxChars,
		areasByID:   areasByID,
		areaIDByName: areaIDByName,
	}
}

func (p *Parser) ParseResume(ctx context.Context, text string) (*model.ResumeProfileDraft, []model.Question, error) {
	cleanedText := p.cleanText(text)
	if len(cleanedText) > p.maxChars {
		cleanedText = cleanedText[:p.maxChars]
	}

	systemPrompt := p.buildParseSystemPrompt()
	userPrompt := fmt.Sprintf("Извлеки структурированные данные из текста резюме:\n\n%s", cleanedText)

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

	p.normalizeDraft(&result.Draft)
	return &result.Draft, p.BuildQuestionsForDraft(&result.Draft), nil
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
	return `Ты помощник по парсингу резюме. Нужно извлечь структурированные данные из текста резюме.

КРИТИЧЕСКИЕ ПРАВИЛА:
1) Верни ТОЛЬКО валидный JSON. Без Markdown, без пояснений, без блоков кода.
2) Заполни ResumeProfileDraft максимально точно по тексту резюме.
3) Для важных полей проставляй confidence в диапазоне 0.0..1.0.
4) Если поле не удалось достоверно определить, оставь его пустым/нулевым и поставь confidence < 0.6.
5) Извлеки 5-10 ключевых навыков в skills_top.
6) Обязательно заполни target_roles (1-3 наиболее вероятные роли), если в резюме есть опыт/должности.
7) professional_role_candidates можно заполнить, если есть разумные альтернативные роли.
8) Сгенерируй 3-4 уточняющих вопроса ТОЛЬКО по полям с низкой уверенностью: target_roles, experience_level, areas, salary_min, work_format.

ПРАВИЛА ДЛЯ РЕГИОНОВ (ОЧЕНЬ ВАЖНО):
- Поле areas должно быть массивом объектов с человекочитаемым названием региона в areas[].name.
- name должен быть текстом (например: "Москва", "Санкт-Петербург", "Казань", "Россия", "Минск"), а НЕ числовым кодом.
- Никогда не записывай цифровой ID в name.
- Если известен только ID региона, положи его в areas[].id, а в name попробуй вывести читаемое название из контекста резюме.
- Если регион совсем не удалось определить, лучше оставить areas пустым, чем возвращать числовой мусор в name.

ПРАВИЛА ДЛЯ ОПЫТА (ОЧЕНЬ ВАЖНО):
- experience_level должен быть только одним из значений HH API:
  - "noExperience" (менее года)
  - "between1And3" (1-3 года)
  - "between3And6" (3-6 лет)
  - "moreThan6" (6+ лет)
- Не возвращай произвольные значения вроде "middle", "junior", "5 years" в поле experience_level.
- Если опыт определить нельзя, верни null.

ПРАВИЛА ДЛЯ ФОРМАТА РАБОТЫ:
- work_format должен содержать только значения из списка: "remote", "hybrid", "office".
- Если формат не определен — верни пустой массив.

ОТВЕТ В ФОРМАТЕ:
{
  "draft": {
    "target_roles": ["Backend Developer", "Go Developer"],
    "professional_role_candidates": [{"id": "...", "name": "...", "confidence": 0.0}],
    "experience_level": null,
    "areas": [{"id": "...", "name": "Москва", "confidence": 0.0}],
    "salary_min": null,
    "currency": null,
    "work_format": [],
    "skills_top": [],
    "notes": null,
    "confidence": {"target_roles": 0.9, "areas": 0.7}
  },
  "questions": [
    {"id": "areas", "text": "Уточните желаемый регион поиска работы", "type": "free_text", "options": []}
  ]
}`
}

func (p *Parser) normalizeDraft(draft *model.ResumeProfileDraft) {
	if draft == nil {
		return
	}

	normalizedAreas := make([]model.Area, 0, len(draft.Areas))
	for i := range draft.Areas {
		area := &draft.Areas[i]
		area.ID = strings.TrimSpace(area.ID)
		area.Name = strings.TrimSpace(area.Name)

		// Частый артефакт LLM: числовой код региона в поле name.
		if isNumericOnly(area.Name) {
			if area.ID == "" {
				area.ID = area.Name
			}
			area.Name = ""
		}

		if area.Name == "" {
			if area.ID != "" {
				if name, ok := p.areasByID[area.ID]; ok {
					area.Name = name
				}
			}
		}

		// Если есть name и нет id, пытаемся привязать к HH-справочнику по имени.
		if area.ID == "" && area.Name != "" {
			key := normalizeAreaName(area.Name)
			if id, ok := p.areaIDByName[key]; ok {
				area.ID = id
			}
		}

		// Жесткая нормализация по требованию:
		// регион должен быть элементом HH-списка, иначе удаляем.
		if area.ID == "" || area.Name == "" {
			continue
		}
		if mappedName, ok := p.areasByID[area.ID]; !ok || mappedName == "" {
			continue
		} else {
			area.Name = mappedName
		}

		normalizedAreas = append(normalizedAreas, *area)
	}
	draft.Areas = dedupeAreas(normalizedAreas)

	normalizedExp := normalizeExperienceLevel(draft.ExperienceLevel)
	draft.ExperienceLevel = normalizedExp
	draft.WorkFormat = normalizeWorkFormatList(draft.WorkFormat)
}

func isNumericOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

type hhAreaNode struct {
	ID    string       `json:"id"`
	Name  string       `json:"name"`
	Areas []hhAreaNode `json:"areas"`
}

func loadHHAreasCatalog() (map[string]string, map[string]string) {
	areasByID := make(map[string]string)
	areaIDByName := make(map[string]string)

	candidates := []string{
		filepath.Clean("hh-data/areas.json"),
		filepath.Clean("../hh-data/areas.json"),
		filepath.Clean("../../hh-data/areas.json"),
	}

	var raw []byte
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err == nil {
			raw = b
			break
		}
	}
	if len(raw) == 0 {
		return areasByID, areaIDByName
	}

	var roots []hhAreaNode
	if err := json.Unmarshal(raw, &roots); err != nil {
		return areasByID, areaIDByName
	}

	var walk func(nodes []hhAreaNode)
	walk = func(nodes []hhAreaNode) {
		for _, n := range nodes {
			id := strings.TrimSpace(n.ID)
			name := strings.TrimSpace(n.Name)
			if id != "" && name != "" {
				areasByID[id] = name
				key := normalizeAreaName(name)
				if _, exists := areaIDByName[key]; !exists {
					areaIDByName[key] = id
				}
			}
			if len(n.Areas) > 0 {
				walk(n.Areas)
			}
		}
	}
	walk(roots)

	return areasByID, areaIDByName
}

func normalizeAreaName(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, "ё", "е")
	v = strings.Join(strings.Fields(v), " ")
	return v
}

func dedupeAreas(in []model.Area) []model.Area {
	out := make([]model.Area, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, a := range in {
		key := a.ID + "|" + normalizeAreaName(a.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, a)
	}
	return out
}

func normalizeExperienceLevel(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.ToLower(strings.TrimSpace(*v))
	if s == "" {
		return nil
	}

	switch s {
	case "noexperience", "between1and3", "between3and6", "morethan6":
		out := canonicalExperienceValue(s)
		return &out
	}

	switch {
	case strings.Contains(s, "без опыта"),
		strings.Contains(s, "менее года"),
		strings.Contains(s, "<1"),
		strings.Contains(s, "0-1"),
		strings.Contains(s, "0–1"):
		out := "noExperience"
		return &out
	case strings.Contains(s, "1-3"),
		strings.Contains(s, "1–3"),
		strings.Contains(s, "1 до 3"):
		out := "between1And3"
		return &out
	case strings.Contains(s, "3-6"),
		strings.Contains(s, "3–6"),
		strings.Contains(s, "3 до 6"):
		out := "between3And6"
		return &out
	case strings.Contains(s, "6+"),
		strings.Contains(s, "более 6"),
		strings.Contains(s, "больше 6"),
		strings.Contains(s, "свыше 6"):
		out := "moreThan6"
		return &out
	default:
		// По требованию: если значение вне допустимого списка — очищаем.
		return nil
	}
}

func canonicalExperienceValue(raw string) string {
	switch raw {
	case "noexperience":
		return "noExperience"
	case "between1and3":
		return "between1And3"
	case "between3and6":
		return "between3And6"
	case "morethan6":
		return "moreThan6"
	default:
		return raw
	}
}

func normalizeWorkFormatList(in []string) []string {
	if len(in) == 0 {
		return in
	}
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, raw := range in {
		s := strings.ToLower(strings.TrimSpace(raw))
		canonical := ""
		switch {
		case strings.Contains(s, "remote"), strings.Contains(s, "удален"):
			canonical = "remote"
		case strings.Contains(s, "hybrid"), strings.Contains(s, "гибрид"):
			canonical = "hybrid"
		case strings.Contains(s, "office"), strings.Contains(s, "офис"):
			canonical = "office"
		}
		if canonical == "" {
			continue
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	return out
}

func (p *Parser) ApplyAnswers(draft *model.ResumeProfileDraft, questions []model.Question, answers []model.QuestionAnswer) (*model.ResumeProfileDraft, []model.Question, string, error) {
	answerMap := make(map[string]string)
	for _, ans := range answers {
		answerMap[ans.QuestionID] = ans.Value
	}

	updatedQuestions := []model.Question{}
	allConfirmed := true

	for _, q := range questions {
		if answer, ok := answerMap[q.ID]; ok {
			answer = strings.TrimSpace(answer)
			applied := false
			switch q.ID {
			case "target_roles":
				if isRelevantFreeText(answer, 2, 80) {
					draft.TargetRoles = []string{answer}
					draft.Confidence["target_roles"] = 1.0
					applied = true
				}
			case "experience_level":
				if normalized := normalizeExperienceLevel(&answer); normalized != nil {
					draft.ExperienceLevel = normalized
					draft.Confidence["experience_level"] = 1.0
					applied = true
				}
			case "salary_min":
				if salary, ok := parseReasonableSalary(answer); ok {
					draft.SalaryMin = &salary
					draft.Confidence["salary_min"] = 1.0
					applied = true
				} else {
					return nil, nil, "", fmt.Errorf("invalid answer for salary_min: expected numeric value in range 1000..10000000")
				}
			case "work_format":
				if wf := normalizeWorkFormatSingle(answer); wf != "" {
					draft.WorkFormat = []string{wf}
					draft.Confidence["work_format"] = 1.0
					applied = true
				}
			case "areas":
				if areas := p.normalizeAreasFromUserAnswer(answer); len(areas) > 0 {
					draft.Areas = areas
					draft.Confidence["areas"] = 1.0
					applied = true
				}
			}
			if !applied {
				updatedQuestions = append(updatedQuestions, q)
				allConfirmed = false
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

	return draft, updatedQuestions, status, nil
}

func isRelevantFreeText(s string, minLen, maxLen int) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	runeLen := len([]rune(s))
	if runeLen < minLen || runeLen > maxLen {
		return false
	}
	low := strings.ToLower(s)
	garbage := []string{"не знаю", "без понятия", "asdf", "qwerty", "123", "нет", "да", "-", "_"}
	for _, g := range garbage {
		if low == g {
			return false
		}
	}
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}
	return hasLetter
}

func parseReasonableSalary(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	s = strings.ReplaceAll(s, "₽", "")
	s = strings.ReplaceAll(s, "руб", "")
	s = strings.ReplaceAll(s, "rur", "")
	s = strings.ReplaceAll(s, "k", "000")
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	// Защита от бессмысленных ответов.
	if v < 1000 || v > 10000000 {
		return 0, false
	}
	return v, true
}

func normalizeWorkFormatSingle(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.Contains(s, "remote"), strings.Contains(s, "удален"):
		return "remote"
	case strings.Contains(s, "hybrid"), strings.Contains(s, "гибрид"):
		return "hybrid"
	case strings.Contains(s, "office"), strings.Contains(s, "офис"):
		return "office"
	default:
		return ""
	}
}

func (p *Parser) BuildQuestionsForDraft(draft *model.ResumeProfileDraft) []model.Question {
	if draft == nil {
		return nil
	}

	out := make([]model.Question, 0, 3)
	if len(draft.Areas) == 0 {
		out = append(out, model.Question{
			ID:      "areas",
			Text:    "Выберите регион поиска работы",
			Type:    "single_choice",
			Options: p.buildAreaQuestionOptions(),
		})
	}
	if len(normalizeWorkFormatList(draft.WorkFormat)) == 0 {
		out = append(out, model.Question{
			ID:      "work_format",
			Text:    "Выберите предпочитаемый формат работы",
			Type:    "single_choice",
			Options: workFormatQuestionOptions,
		})
	}
	if normalizeExperienceLevel(draft.ExperienceLevel) == nil {
		out = append(out, model.Question{
			ID:      "experience_level",
			Text:    "Укажите ваш опыт работы",
			Type:    "single_choice",
			Options: experienceQuestionOptions,
		})
	}
	if draft.SalaryMin == nil || *draft.SalaryMin <= 0 {
		out = append(out, model.Question{
			ID:      "salary_min",
			Text:    "Укажите ожидаемую зарплату в рублях (число без пробелов), либо выберите один из вариантов",
			Type:    "numeric_input",
			Options: salaryQuestionPresets,
		})
	}
	return out
}

func (p *Parser) buildAreaQuestionOptions() []string {
	out := make([]string, 0, len(defaultAreaQuestionOptions))
	for _, name := range defaultAreaQuestionOptions {
		if _, ok := p.areaIDByName[normalizeAreaName(name)]; ok {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		out = append(out, "Россия")
	}
	return out
}

func (p *Parser) normalizeAreasFromUserAnswer(answer string) []model.Area {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return nil
	}

	parts := strings.FieldsFunc(answer, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})

	out := make([]model.Area, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		key := normalizeAreaName(name)
		id, ok := p.areaIDByName[key]
		if !ok {
			continue
		}
		canonicalName := p.areasByID[id]
		dedupeKey := id + "|" + canonicalName
		if _, exists := seen[dedupeKey]; exists {
			continue
		}
		seen[dedupeKey] = struct{}{}
		out = append(out, model.Area{
			ID:         id,
			Name:       canonicalName,
			Confidence: 1.0,
		})
	}
	return out
}
