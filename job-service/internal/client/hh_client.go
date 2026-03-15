package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type HHVacancy struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Snippet     *HHSnippet   `json:"snippet"`
	Salary      *HHSalary    `json:"salary"`
	Employer    HHEmployer   `json:"employer"`
	Area        HHArea       `json:"area"`
	URL         string       `json:"alternate_url"`
	Experience  HHExperience `json:"experience"`
	Archived    bool         `json:"archived"`
}

type HHSnippet struct {
	Requirement    *string `json:"requirement"`
	Responsibility *string `json:"responsibility"`
}

// GetDescription returns responsibility from snippet (search results)
// or description (full vacancy endpoint), whichever is available.
func (v *HHVacancy) GetDescription() string {
	if v.Snippet != nil && v.Snippet.Responsibility != nil && *v.Snippet.Responsibility != "" {
		return *v.Snippet.Responsibility
	}
	return v.Description
}

type HHExperience struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type HHSalary struct {
	From     *int    `json:"from"`
	To       *int    `json:"to"`
	Currency string  `json:"currency"`
}

type HHEmployer struct {
	Name     string      `json:"name"`
	LogoURLs *HHLogoURLs `json:"logo_urls"`
}

type HHLogoURLs struct {
	Size90  string `json:"90"`
	Size240 string `json:"240"`
	Original string `json:"original"`
}

type HHArea struct {
	Name string `json:"name"`
}

type HHResponse struct {
	Items []HHVacancy `json:"items"`
	Found int         `json:"found"`
	Pages int         `json:"pages"`
	Page  int         `json:"page"`
}

type HHClient struct {
	host       string
	appToken   string
	userAgent  string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewHHClient(host, appToken, userAgent string, timeout time.Duration, logger *slog.Logger) *HHClient {
	return &HHClient{
		host:       host,
		appToken:   appToken,
		userAgent:  userAgent,
		httpClient: &http.Client{Timeout: timeout},
		logger:     logger,
	}
}

func (c *HHClient) SearchVacancies(ctx context.Context, params map[string]string) (*HHResponse, error) {
	u := fmt.Sprintf("https://%s/vacancies", c.host)
	reqURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	q := reqURL.Query()
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	reqURL.RawQuery = q.Encode()
	fullURL := reqURL.String()

	if c.logger != nil {
		c.logger.Info("HH request", "url", fullURL, "params", params)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("HH-User-Agent", c.userAgent)
	if c.appToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.appToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search vacancies: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HH response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		return nil, fmt.Errorf("hh.ru returned %d: %s", resp.StatusCode, bodyStr)
	}

	var hhResp HHResponse
	if err := json.Unmarshal(body, &hhResp); err != nil {
		return nil, fmt.Errorf("failed to decode HH response: %w", err)
	}

	if c.logger != nil {
		c.logger.Info("HH response", "found", hhResp.Found, "pages", hhResp.Pages, "page", hhResp.Page, "items_count", len(hhResp.Items))
	}

	return &hhResp, nil
}

// GetVacancyByID fetches a single vacancy by ID from HH API.
func (c *HHClient) GetVacancyByID(ctx context.Context, vacancyID string) (*HHVacancy, error) {
	if vacancyID == "" {
		return nil, fmt.Errorf("vacancy_id is required")
	}

	u := fmt.Sprintf("https://%s/vacancies/%s", c.host, vacancyID)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("HH-User-Agent", c.userAgent)
	if c.appToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.appToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vacancy: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HH response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("vacancy not found: %s", vacancyID)
	}
	if resp.StatusCode != http.StatusOK {
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		return nil, fmt.Errorf("hh.ru returned %d: %s", resp.StatusCode, bodyStr)
	}

	var v HHVacancy
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, fmt.Errorf("failed to decode vacancy: %w", err)
	}
	return &v, nil
}

// professional_role=96 — "Программист, разработчик" в справочнике HH (как в рабочем примере).
const hhProfessionalRoleDeveloper = "96"

func BuildHHQuery(profile *ResumeProfile, page, perPage int) map[string]string {
	params := make(map[string]string)

	// Как в примере: короткий text (одна роль + до 2 навыков), обязательный professional_role для IT-поиска
	if len(profile.TargetRoles) > 0 {
		textParts := []string{profile.TargetRoles[0]}
		if len(profile.SkillsTop) > 0 {
			n := 2
			if len(profile.SkillsTop) < n {
				n = len(profile.SkillsTop)
			}
			textParts = append(textParts, strings.Join(profile.SkillsTop[:n], " "))
		}
		params["text"] = strings.TrimSpace(strings.Join(textParts, " "))
		params["professional_role"] = hhProfessionalRoleDeveloper
	}

	if profile.ExperienceLevel != nil {
		if v := mapExperienceLevel(*profile.ExperienceLevel); v != "" {
			params["experience"] = v
		}
	}

	if len(profile.Areas) > 0 {
		areaIDs := validHHAreaIDs(profile.Areas)
		if len(areaIDs) > 0 {
			// HH API accepts only a single area ID per request
			params["area"] = areaIDs[0]
		}
	}

	if profile.SalaryMin != nil {
		params["salary"] = strconv.FormatFloat(*profile.SalaryMin, 'f', 0, 64)
		params["only_with_salary"] = "true"
		if profile.Currency != nil && *profile.Currency != "" {
			params["currency"] = *profile.Currency
		} else {
			params["currency"] = "RUR"
		}
	}

	if len(profile.WorkFormat) > 0 {
		if v := mapWorkFormat(profile.WorkFormat[0]); v != "" {
			params["schedule"] = v
		}
	}

	params["per_page"] = strconv.Itoa(perPage)
	params["page"] = strconv.Itoa(page)

	return params
}

// validHHAreaIDs returns only area IDs that are valid for HH API (numeric IDs from HH areas tree).
// Non-numeric or empty IDs are skipped to avoid "bad argument: area" from the API.
func validHHAreaIDs(areas []Area) []string {
	out := make([]string, 0, len(areas))
	for _, a := range areas {
		id := strings.TrimSpace(a.ID)
		if id == "" {
			continue
		}
		// HH area parameter accepts only numeric IDs (e.g. 1=Москва, 2=СПб)
		allNumeric := true
		for _, c := range id {
			if c < '0' || c > '9' {
				allNumeric = false
				break
			}
		}
		if allNumeric {
			out = append(out, id)
		}
	}
	return out
}

func mapExperienceLevel(level string) string {
	level = strings.ToLower(level)
	switch {
	case strings.Contains(level, "junior") || strings.Contains(level, "начал"):
		return "between1And3"
	case strings.Contains(level, "middle") || strings.Contains(level, "средн"):
		return "between3And6"
	case strings.Contains(level, "senior") || strings.Contains(level, "старш"):
		return "moreThan6"
	default:
		return ""
	}
}

func mapWorkFormat(format string) string {
	format = strings.ToLower(format)
	switch {
	case strings.Contains(format, "remote") || strings.Contains(format, "удален"):
		return "remote"
	case strings.Contains(format, "hybrid") || strings.Contains(format, "гибрид"):
		return "flexible"
	case strings.Contains(format, "office") || strings.Contains(format, "офис"):
		return "fullDay"
	default:
		return ""
	}
}
