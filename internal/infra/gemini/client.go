package gemini

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

type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
	model      string
}

func NewClient(apiKey, model string) *Client {
	return NewClientWithURL(apiKey, model, "https://generativelanguage.googleapis.com/v1beta")
}

func NewClientWithURL(apiKey, model, baseURL string) *Client {
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		model:      model,
	}
}

type content struct {
	Parts []part `json:"parts"`
	Role  string `json:"role,omitempty"`
}

type part struct {
	Text string `json:"text"`
}

type request struct {
	Contents         []content        `json:"contents"`
	SystemInstruct   *content         `json:"systemInstruction,omitempty"`
	GenerationConfig generationConfig `json:"generationConfig"`
}

type generationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float64 `json:"temperature"`
}

type response struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

type parsedIntent struct {
	Action     string         `json:"action"`
	TargetName string         `json:"target_name"`
	TargetType string         `json:"target_type"`
	Parameters map[string]any `json:"parameters"`
	Confidence float64        `json:"confidence"`
}

func (c *Client) Parse(ctx context.Context, text string, registry application.DeviceRegistry) (*domain.Command, error) {
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
		SystemInstruct: &content{
			Parts: []part{{Text: systemPrompt}},
		},
		Contents: []content{
			{
				Role:  "user",
				Parts: []part{{Text: text}},
			},
		},
		GenerationConfig: generationConfig{
			MaxOutputTokens: 256,
			Temperature:     0.1,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var result response
	retryErr := infra.WithRetry(ctx, infra.DefaultRetryConfig(), func() error {
		url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("sending request: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			if infra.IsRetryableHTTPStatus(resp.StatusCode) {
				return fmt.Errorf("gemini API error %d: %s (retryable)", resp.StatusCode, string(respBody))
			}
			return fmt.Errorf("gemini API error %d: %s", resp.StatusCode, string(respBody))
		}

		if err = json.Unmarshal(respBody, &result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}

		return nil
	})

	if retryErr != nil {
		return nil, retryErr
	}

	if result.Error != nil {
		return nil, fmt.Errorf("gemini error: %s", result.Error.Message)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from gemini")
	}

	responseText := strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)
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
