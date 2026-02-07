package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"career-coach-service/internal/model"
)

type Client struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

func NewClient(apiKey, endpoint string, timeout time.Duration) *Client {
	return &Client{
		apiKey:   apiKey,
		endpoint: endpoint,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) ChatCompletion(ctx context.Context, modelName string, messages []model.Message) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("openrouter API key is empty")
	}

	var reqBody model.OpenRouterRequest
	reqBody.Model = modelName
	reqBody.Messages = messages

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/your-org/your-repo") // Optional: for OpenRouter analytics
	req.Header.Set("X-Title", "Interview Prep App") // Optional: for OpenRouter analytics

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp model.OpenRouterResponse
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != nil {
			return "", fmt.Errorf("openrouter API error: %s (type: %s)", errorResp.Error.Message, errorResp.Error.Type)
		}
		return "", fmt.Errorf("openrouter API returned status %d: %s", resp.StatusCode, string(body))
	}

	var openRouterResp model.OpenRouterResponse
	if err := json.Unmarshal(body, &openRouterResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return openRouterResp.Choices[0].Message.Content, nil
}
