package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"smart-home/internal/domain"
	"smart-home/internal/infra"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	// Remove trailing slash if present
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Entity represents a Home Assistant entity
type Entity struct {
	EntityID    string                 `json:"entity_id"`
	State       string                 `json:"state"`
	Attributes  map[string]interface{} `json:"attributes"`
	LastChanged string                 `json:"last_changed"`
}

// Scene represents a Home Assistant scene
type Scene struct {
	EntityID   string                 `json:"entity_id"`
	State      string                 `json:"state"`
	Attributes map[string]interface{} `json:"attributes"`
}

func (c *Client) ExecuteCommand(ctx context.Context, cmd *domain.Command) error {
	service, data := c.buildServiceCall(cmd)
	if service == "" {
		return fmt.Errorf("unknown action: %s", cmd.Action)
	}

	// Split service into domain and service name (e.g., "light.turn_on" -> "light", "turn_on")
	parts := strings.SplitN(service, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid service format: %s", service)
	}

	path := fmt.Sprintf("/api/services/%s/%s", parts[0], parts[1])

	// Add entity_id to data
	data["entity_id"] = cmd.TargetID

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	_, err = c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return fmt.Errorf("executing command: %w", err)
	}

	return nil
}

func (c *Client) TriggerScene(ctx context.Context, sceneID string) error {
	path := "/api/services/scene/turn_on"
	data := map[string]interface{}{
		"entity_id": sceneID,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	_, err = c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return fmt.Errorf("triggering scene: %w", err)
	}

	return nil
}

func (c *Client) buildServiceCall(cmd *domain.Command) (string, map[string]interface{}) {
	data := make(map[string]interface{})

	// Determine entity domain from entity_id (e.g., "light.living_room" -> "light")
	entityDomain := "light" // default
	if parts := strings.SplitN(cmd.TargetID, ".", 2); len(parts) == 2 {
		entityDomain = parts[0]
	}

	switch cmd.Action {
	case domain.ActionTurnOn:
		return entityDomain + ".turn_on", data

	case domain.ActionTurnOff:
		return entityDomain + ".turn_off", data

	case domain.ActionSetLevel:
		level, ok := cmd.Parameters["level"].(float64)
		if !ok {
			level = 100
		}
		// Home Assistant uses brightness 0-255
		data["brightness"] = int(level * 2.55)
		return "light.turn_on", data

	case domain.ActionSetColor:
		// Home Assistant accepts various color formats
		if color, ok := cmd.Parameters["color"].(string); ok {
			data["color_name"] = color
		}
		return "light.turn_on", data

	default:
		return "", nil
	}
}

func (c *Client) GetDevices(ctx context.Context) ([]domain.Device, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/states", nil)
	if err != nil {
		return nil, fmt.Errorf("fetching states: %w", err)
	}

	var entities []Entity
	if err := json.Unmarshal(resp, &entities); err != nil {
		return nil, fmt.Errorf("parsing states: %w", err)
	}

	devices := make([]domain.Device, 0)
	for _, e := range entities {
		deviceType := entityDomainToDeviceType(e.EntityID)
		if deviceType == "" {
			continue // Skip non-device entities
		}

		name := e.EntityID
		if friendlyName, ok := e.Attributes["friendly_name"].(string); ok {
			name = friendlyName
		}

		devices = append(devices, domain.Device{
			ID:       e.EntityID,
			Name:     name,
			Category: strings.SplitN(e.EntityID, ".", 2)[0],
			Type:     deviceType,
			Online:   e.State != "unavailable",
		})
	}

	return devices, nil
}

func (c *Client) GetScenes(ctx context.Context) ([]domain.Scene, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/states", nil)
	if err != nil {
		return nil, fmt.Errorf("fetching states: %w", err)
	}

	var entities []Entity
	if err := json.Unmarshal(resp, &entities); err != nil {
		return nil, fmt.Errorf("parsing states: %w", err)
	}

	scenes := make([]domain.Scene, 0)
	for _, e := range entities {
		if !strings.HasPrefix(e.EntityID, "scene.") {
			continue
		}

		name := e.EntityID
		if friendlyName, ok := e.Attributes["friendly_name"].(string); ok {
			name = friendlyName
		}

		scenes = append(scenes, domain.Scene{
			ID:     e.EntityID,
			Name:   name,
			Status: e.State,
		})
	}

	return scenes, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	var respBody []byte

	retryErr := infra.WithRetry(ctx, infra.DefaultRetryConfig(), func() error {
		var bodyReader io.Reader
		if body != nil {
			bodyReader = strings.NewReader(string(body))
		}

		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("sending request: %w", err)
		}
		defer resp.Body.Close()

		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("unauthorized: check your Home Assistant token")
		}

		if infra.IsRetryableHTTPStatus(resp.StatusCode) {
			return fmt.Errorf("home assistant API error %d (retryable): %s", resp.StatusCode, string(respBody))
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("home assistant API error %d: %s", resp.StatusCode, string(respBody))
		}

		return nil
	})

	if retryErr != nil {
		return nil, retryErr
	}

	return respBody, nil
}

func entityDomainToDeviceType(entityID string) domain.DeviceType {
	parts := strings.SplitN(entityID, ".", 2)
	if len(parts) != 2 {
		return ""
	}

	switch parts[0] {
	case "light":
		return domain.DeviceTypeLight
	case "switch":
		return domain.DeviceTypeSwitch
	case "climate":
		return domain.DeviceTypeThermostat
	case "binary_sensor", "sensor":
		return domain.DeviceTypeSensor
	case "fan":
		return domain.DeviceTypeOther
	default:
		return "" // Skip unknown entity types
	}
}
