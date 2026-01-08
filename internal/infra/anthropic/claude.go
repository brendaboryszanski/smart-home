package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"smart-home/internal/application"
	"smart-home/internal/domain"
	"smart-home/internal/infra"
)

type ClaudeClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
	model      string
}

func NewClaudeClient(apiKey, model string) *ClaudeClient {
	return NewClaudeClientWithURL(apiKey, model, "https://api.anthropic.com/v1")
}

func NewClaudeClientWithURL(apiKey, model, baseURL string) *ClaudeClient {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &ClaudeClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		model:      model,
	}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
}

type response struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

type parsedIntent struct {
	Action     string         `json:"action"`
	TargetName string         `json:"target_name"`
	TargetType string         `json:"target_type"`
	Parameters map[string]any `json:"parameters"`
	Confidence float64        `json:"confidence"`
}

func (c *ClaudeClient) Parse(ctx context.Context, text string, registry application.DeviceRegistry) (*domain.Command, error) {
	systemPrompt := fmt.Sprintf(`You are a smart home assistant. Your task is to interpret voice commands and extract the intent.

%s

IMPORTANT:
- If the user mentions a scene, use target_type "scene"
- If the user mentions a device, use target_type "device"
- Use the EXACT name of the device or scene as it appears in the list
- If you don't understand the command, use action "unknown"
- The user may speak in English or Spanish, understand both

Respond ONLY with valid JSON (no markdown, no backticks):
{
  "action": "turn_on|turn_off|set_level|set_color|run_scene|get_status|unknown",
  "target_name": "exact device or scene name",
  "target_type": "device|scene",
  "parameters": {"level": 50, "color": "red"},
  "confidence": 0.95
}`, registry.Summary())

	reqBody := request{
		Model:     c.model,
		MaxTokens: 256,
		System:    systemPrompt,
		Messages: []message{
			{Role: "user", Content: text},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var result response
	retryErr := infra.WithRetry(ctx, infra.DefaultRetryConfig(), func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("sending request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			if infra.IsRetryableHTTPStatus(resp.StatusCode) {
				return fmt.Errorf("claude API error %d: %s (retryable)", resp.StatusCode, string(respBody))
			}
			return fmt.Errorf("claude API error %d: %s", resp.StatusCode, string(respBody))
		}

		if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}

		return nil
	})

	if retryErr != nil {
		return nil, retryErr
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("empty response from claude")
	}

	responseText := strings.TrimSpace(result.Content[0].Text)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	var intent parsedIntent
	if err = json.Unmarshal([]byte(responseText), &intent); err != nil {
		return nil, fmt.Errorf("parsing intent JSON (%s): %w", responseText, err)
	}

	return &domain.Command{
		Action:     domain.Action(intent.Action),
		TargetName: intent.TargetName,
		TargetType: domain.TargetType(intent.TargetType),
		Parameters: intent.Parameters,
		RawText:    text,
		Confidence: intent.Confidence,
	}, nil
}

