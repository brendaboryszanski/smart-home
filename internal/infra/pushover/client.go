package pushover

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	token      string
	userKey    string
	httpClient *http.Client
}

func NewClient(token, userKey string) *Client {
	return &Client{
		token:      token,
		userKey:    userKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Notify(ctx context.Context, message string) error {
	if c.token == "" || c.userKey == "" {
		return nil
	}

	data := url.Values{}
	data.Set("token", c.token)
	data.Set("user", c.userKey)
	data.Set("message", message)
	data.Set("title", "Smart Home")

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.pushover.net/1/messages.json",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pushover error: %s", resp.Status)
	}

	return nil
}

