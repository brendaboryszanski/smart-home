package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"smart-home/internal/infra"
)

type WhisperClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
	language   string
}

func NewWhisperClient(apiKey, language string) *WhisperClient {
	return &WhisperClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.openai.com/v1",
		language:   language,
	}
}

type transcriptionResponse struct {
	Text string `json:"text"`
}

func (c *WhisperClient) Transcribe(ctx context.Context, audio []byte) (string, error) {
	var result transcriptionResponse

	retryErr := infra.WithRetry(ctx, infra.DefaultRetryConfig(), func() error {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", "audio.wav")
		if err != nil {
			return fmt.Errorf("creating form file: %w", err)
		}

		if _, err = part.Write(audio); err != nil {
			return fmt.Errorf("writing audio: %w", err)
		}

		if err = writer.WriteField("model", "whisper-1"); err != nil {
			return fmt.Errorf("writing model field: %w", err)
		}

		if err = writer.WriteField("language", c.language); err != nil {
			return fmt.Errorf("writing language field: %w", err)
		}

		if err = writer.Close(); err != nil {
			return fmt.Errorf("closing writer: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/audio/transcriptions", body)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("sending request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			if infra.IsRetryableHTTPStatus(resp.StatusCode) {
				return fmt.Errorf("whisper API error %d: %s (retryable)", resp.StatusCode, string(respBody))
			}
			return fmt.Errorf("whisper API error %d: %s", resp.StatusCode, string(respBody))
		}

		if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}

		return nil
	})

	if retryErr != nil {
		return "", retryErr
	}

	return result.Text, nil
}

