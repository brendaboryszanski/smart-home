package anthropic_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"smart-home/internal/domain"
	"smart-home/internal/infra/anthropic"
)

type mockRegistry struct{}

func (m *mockRegistry) Sync(_ context.Context) error           { return nil }
func (m *mockRegistry) GetDevices() []domain.Device            { return nil }
func (m *mockRegistry) GetScenes() []domain.Scene              { return nil }
func (m *mockRegistry) FindDeviceByName(_ string) (*domain.Device, bool) { return nil, false }
func (m *mockRegistry) FindSceneByName(_ string) (*domain.Scene, bool)   { return nil, false }
func (m *mockRegistry) Summary() string {
	return `## Dispositivos:
- Luz Living (tipo: light)
- Luz Cocina (tipo: light)
## Escenas:
- Buenas Noches`
}

func TestClaudeClient_Parse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		response := map[string]any{
			"content": []map[string]string{
				{"text": `{"action":"turn_on","target_name":"Luz Living","target_type":"device","parameters":{},"confidence":0.95}`},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := anthropic.NewClaudeClientWithURL("test-key", "claude-test", server.URL)

	cmd, err := client.Parse(context.Background(), "prende la luz del living", &mockRegistry{})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if cmd.Action != domain.ActionTurnOn {
		t.Errorf("Action: got %s, want turn_on", cmd.Action)
	}

	if cmd.TargetName != "Luz Living" {
		t.Errorf("TargetName: got %s, want Luz Living", cmd.TargetName)
	}

	if cmd.TargetType != domain.TargetTypeDevice {
		t.Errorf("TargetType: got %s, want device", cmd.TargetType)
	}
}

func TestClaudeClient_ParseScene(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"content": []map[string]string{
				{"text": `{"action":"run_scene","target_name":"Buenas Noches","target_type":"scene","parameters":{},"confidence":0.98}`},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := anthropic.NewClaudeClientWithURL("test-key", "claude-test", server.URL)

	cmd, err := client.Parse(context.Background(), "activa la escena buenas noches", &mockRegistry{})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if cmd.Action != domain.ActionRunScene {
		t.Errorf("Action: got %s, want run_scene", cmd.Action)
	}

	if cmd.TargetType != domain.TargetTypeScene {
		t.Errorf("TargetType: got %s, want scene", cmd.TargetType)
	}
}

func TestClaudeClient_ParseUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"content": []map[string]string{
				{"text": `{"action":"unknown","target_name":"","target_type":"","parameters":{},"confidence":0.1}`},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := anthropic.NewClaudeClientWithURL("test-key", "claude-test", server.URL)

	cmd, err := client.Parse(context.Background(), "qué hora es", &mockRegistry{})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if cmd.Action != domain.ActionUnknown {
		t.Errorf("Action: got %s, want unknown", cmd.Action)
	}
}

