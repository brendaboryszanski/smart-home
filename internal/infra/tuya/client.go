package tuya

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"smart-home/internal/domain"
	"smart-home/internal/infra"
)

type Client struct {
	clientID   string
	secret     string
	baseURL    string
	httpClient *http.Client

	mu       sync.RWMutex
	token    string
	expireAt time.Time
	uid      string
}

func NewClient(clientID, secret, region string) *Client {
	baseURL := "https://openapi.tuyaus.com"
	switch strings.ToLower(region) {
	case "eu":
		baseURL = "https://openapi.tuyaeu.com"
	case "cn":
		baseURL = "https://openapi.tuyacn.com"
	case "in":
		baseURL = "https://openapi.tuyain.com"
	}

	return NewClientWithURL(clientID, secret, baseURL)
}

func NewClientWithURL(clientID, secret, baseURL string) *Client {
	return &Client{
		clientID:   clientID,
		secret:     secret,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) ExecuteCommand(ctx context.Context, cmd *domain.Command) error {
	commands := c.buildCommands(cmd)
	body, _ := json.Marshal(map[string]any{"commands": commands})

	path := fmt.Sprintf("/v1.0/iot-03/devices/%s/commands", cmd.TargetID)
	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return fmt.Errorf("executing command: %w", err)
	}

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("tuya error: %s", result.Msg)
	}

	return nil
}

func (c *Client) TriggerScene(ctx context.Context, sceneID string) error {
	path := fmt.Sprintf("/v1.0/iot-03/scenes/%s/trigger", sceneID)
	resp, err := c.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("triggering scene: %w", err)
	}

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("tuya error: %s", result.Msg)
	}

	return nil
}

func (c *Client) buildCommands(cmd *domain.Command) []map[string]any {
	switch cmd.Action {
	case domain.ActionTurnOn:
		return []map[string]any{{"code": "switch_led", "value": true}}
	case domain.ActionTurnOff:
		return []map[string]any{{"code": "switch_led", "value": false}}
	case domain.ActionSetLevel:
		level, ok := cmd.Parameters["level"].(float64)
		if !ok {
			level = 100
		}
		tuyaLevel := int(level * 10)
		return []map[string]any{
			{"code": "switch_led", "value": true},
			{"code": "bright_value_v2", "value": tuyaLevel},
		}
	case domain.ActionSetColor:
		return []map[string]any{{"code": "switch_led", "value": true}}
	default:
		return []map[string]any{}
	}
}

func (c *Client) GetDevices(ctx context.Context) ([]domain.Device, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/v1.0/iot-01/associated-users/devices", nil)
	if err != nil {
		return nil, fmt.Errorf("fetching devices: %w", err)
	}

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		Result  struct {
			Devices []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Category string `json:"category"`
				Online   bool   `json:"online"`
			} `json:"devices"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing devices: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("tuya error: %s", result.Msg)
	}

	devices := make([]domain.Device, 0, len(result.Result.Devices))
	for _, d := range result.Result.Devices {
		devices = append(devices, domain.Device{
			ID:       d.ID,
			Name:     d.Name,
			Category: d.Category,
			Type:     categoryToType(d.Category),
			Online:   d.Online,
		})
	}

	return devices, nil
}

func (c *Client) GetHomes(ctx context.Context) ([]string, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1.0/users/%s/homes", c.uid)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching homes: %w", err)
	}

	var result struct {
		Success bool `json:"success"`
		Result  []struct {
			HomeID int64 `json:"home_id"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing homes: %w", err)
	}

	homes := make([]string, 0, len(result.Result))
	for _, h := range result.Result {
		homes = append(homes, fmt.Sprintf("%d", h.HomeID))
	}

	return homes, nil
}

func (c *Client) GetScenes(ctx context.Context, homeID string) ([]domain.Scene, error) {
	path := fmt.Sprintf("/v1.0/homes/%s/scenes", homeID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching scenes: %w", err)
	}

	var result struct {
		Success bool `json:"success"`
		Result  []struct {
			SceneID string `json:"scene_id"`
			Name    string `json:"name"`
			Status  string `json:"status"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing scenes: %w", err)
	}

	scenes := make([]domain.Scene, 0, len(result.Result))
	for _, s := range result.Result {
		scenes = append(scenes, domain.Scene{
			ID:     s.SceneID,
			Name:   s.Name,
			HomeID: homeID,
			Status: s.Status,
		})
	}

	return scenes, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}

	var respBody []byte
	retryErr := infra.WithRetry(ctx, infra.DefaultRetryConfig(), func() error {
		timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())

		var bodyReader io.Reader
		if body != nil {
			bodyReader = strings.NewReader(string(body))
		}

		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		sign := c.calcSign(timestamp, c.token, method, path, body)

		req.Header.Set("client_id", c.clientID)
		req.Header.Set("access_token", c.token)
		req.Header.Set("sign", sign)
		req.Header.Set("t", timestamp)
		req.Header.Set("sign_method", "HMAC-SHA256")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("sending request: %w", err)
		}
		defer resp.Body.Close()

		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		// Check if we should retry based on status code
		if infra.IsRetryableHTTPStatus(resp.StatusCode) {
			return fmt.Errorf("tuya API error %d (retryable): %s", resp.StatusCode, string(respBody))
		}

		return nil
	})

	if retryErr != nil {
		return nil, retryErr
	}

	return respBody, nil
}

func (c *Client) ensureToken(ctx context.Context) error {
	c.mu.RLock()
	if c.token != "" && time.Now().Add(5*time.Minute).Before(c.expireAt) {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Add(5*time.Minute).Before(c.expireAt) {
		return nil
	}

	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	path := "/v1.0/token?grant_type=1"
	sign := c.calcSign(timestamp, "", http.MethodGet, path, nil)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating token request: %w", err)
	}

	req.Header.Set("client_id", c.clientID)
	req.Header.Set("sign", sign)
	req.Header.Set("t", timestamp)
	req.Header.Set("sign_method", "HMAC-SHA256")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading token response: %w", err)
	}

	var tokenResp struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		Result  struct {
			AccessToken string `json:"access_token"`
			ExpireTime  int64  `json:"expire_time"`
			UID         string `json:"uid"`
		} `json:"result"`
	}

	if err = json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("parsing token response: %w", err)
	}

	if !tokenResp.Success {
		return fmt.Errorf("token error: %s", tokenResp.Msg)
	}

	c.token = tokenResp.Result.AccessToken
	c.expireAt = time.Now().Add(time.Duration(tokenResp.Result.ExpireTime) * time.Second)
	c.uid = tokenResp.Result.UID

	return nil
}

func (c *Client) calcSign(timestamp, token, method, path string, body []byte) string {
	str := c.clientID + token + timestamp + c.stringToSign(method, path, body)
	h := hmac.New(sha256.New, []byte(c.secret))
	h.Write([]byte(str))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}

func (c *Client) stringToSign(method, path string, body []byte) string {
	bodyHash := sha256.Sum256(body)
	return method + "\n" + hex.EncodeToString(bodyHash[:]) + "\n\n" + path
}

func categoryToType(category string) domain.DeviceType {
	switch category {
	case "dj", "dd", "fwd", "xdd", "dc", "tgq":
		return domain.DeviceTypeLight
	case "cz", "pc":
		return domain.DeviceTypePlug
	case "kg", "tdq":
		return domain.DeviceTypeSwitch
	case "wk", "wkf":
		return domain.DeviceTypeThermostat
	case "pir", "mcs", "ywbj", "rqbj", "jwbj":
		return domain.DeviceTypeSensor
	default:
		return domain.DeviceTypeOther
	}
}

