package tuya_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"smart-home/internal/domain"
	"smart-home/internal/infra/tuya"
)

func TestClient_GetDevices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1.0/token":
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"result": map[string]any{
					"access_token": "test-token",
					"expire_time":  7200,
					"uid":          "test-uid",
				},
			})
		case "/v1.0/iot-01/associated-users/devices":
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"result": map[string]any{
					"devices": []map[string]any{
						{"id": "dev1", "name": "Luz Living", "category": "dj", "online": true},
						{"id": "dev2", "name": "Enchufe Cocina", "category": "cz", "online": true},
					},
				},
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := tuya.NewClientWithURL("client-id", "secret", server.URL)

	devices, err := client.GetDevices(context.Background())
	if err != nil {
		t.Fatalf("GetDevices error: %v", err)
	}

	if len(devices) != 2 {
		t.Errorf("devices count: got %d, want 2", len(devices))
	}

	if devices[0].Name != "Luz Living" {
		t.Errorf("device name: got %s, want Luz Living", devices[0].Name)
	}

	if devices[0].Type != domain.DeviceTypeLight {
		t.Errorf("device type: got %s, want light", devices[0].Type)
	}
}

func TestClient_ExecuteCommand(t *testing.T) {
	commandReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1.0/token":
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"result": map[string]any{
					"access_token": "test-token",
					"expire_time":  7200,
					"uid":          "test-uid",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1.0/iot-03/devices/dev1/commands":
			commandReceived = true
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := tuya.NewClientWithURL("client-id", "secret", server.URL)

	cmd := &domain.Command{
		Action:   domain.ActionTurnOn,
		TargetID: "dev1",
	}

	err := client.ExecuteCommand(context.Background(), cmd)
	if err != nil {
		t.Fatalf("ExecuteCommand error: %v", err)
	}

	if !commandReceived {
		t.Error("command was not sent to server")
	}
}

func TestClient_TriggerScene(t *testing.T) {
	sceneTriggered := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1.0/token":
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"result": map[string]any{
					"access_token": "test-token",
					"expire_time":  7200,
					"uid":          "test-uid",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1.0/iot-03/scenes/scene1/trigger":
			sceneTriggered = true
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := tuya.NewClientWithURL("client-id", "secret", server.URL)

	err := client.TriggerScene(context.Background(), "scene1")
	if err != nil {
		t.Fatalf("TriggerScene error: %v", err)
	}

	if !sceneTriggered {
		t.Error("scene was not triggered")
	}
}

