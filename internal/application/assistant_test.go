package application_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"smart-home/internal/application"
	"smart-home/internal/domain"
)

type mockAudioSource struct {
	commands [][]byte
	index    int
}

func (m *mockAudioSource) Start(_ context.Context) error { return nil }
func (m *mockAudioSource) Stop() error                   { return nil }
func (m *mockAudioSource) Name() string                  { return "mock" }

func (m *mockAudioSource) NextCommand(_ context.Context) ([]byte, error) {
	if m.index >= len(m.commands) {
		return nil, context.Canceled
	}
	audio := m.commands[m.index]
	m.index++
	return audio, nil
}

type mockSTT struct {
	transcriptions map[string]string
}

func (m *mockSTT) Transcribe(_ context.Context, audio []byte) (string, error) {
	key := string(audio)
	if text, ok := m.transcriptions[key]; ok {
		return text, nil
	}
	return "unknown command", nil
}

type mockIntentParser struct {
	intents map[string]*domain.Command
}

func (m *mockIntentParser) Parse(_ context.Context, text string, _ application.DeviceRegistry) (*domain.Command, error) {
	if cmd, ok := m.intents[text]; ok {
		return cmd, nil
	}
	return &domain.Command{Action: domain.ActionUnknown}, nil
}

type mockDeviceController struct {
	executedCommands []*domain.Command
	triggeredScenes  []string
	doneChan         chan struct{}
	expectedCommands int
}

func (m *mockDeviceController) ExecuteCommand(_ context.Context, cmd *domain.Command) error {
	m.executedCommands = append(m.executedCommands, cmd)
	if m.doneChan != nil && len(m.executedCommands) >= m.expectedCommands {
		close(m.doneChan)
	}
	return nil
}

func (m *mockDeviceController) TriggerScene(_ context.Context, sceneID string) error {
	m.triggeredScenes = append(m.triggeredScenes, sceneID)
	if m.doneChan != nil && len(m.triggeredScenes) >= m.expectedCommands {
		close(m.doneChan)
	}
	return nil
}

type mockRegistry struct {
	devices []domain.Device
	scenes  []domain.Scene
}

func (m *mockRegistry) Sync(_ context.Context) error { return nil }
func (m *mockRegistry) GetDevices() []domain.Device  { return m.devices }
func (m *mockRegistry) GetScenes() []domain.Scene    { return m.scenes }
func (m *mockRegistry) Summary() string              { return "mock devices" }

func (m *mockRegistry) FindDeviceByName(name string) (*domain.Device, bool) {
	for i, d := range m.devices {
		if d.Name == name {
			return &m.devices[i], true
		}
	}
	return nil, false
}

func (m *mockRegistry) FindSceneByName(name string) (*domain.Scene, bool) {
	for i, s := range m.scenes {
		if s.Name == name {
			return &m.scenes[i], true
		}
	}
	return nil, false
}

func (m *mockRegistry) StartPeriodicSync(_ context.Context, _ time.Duration) {}

func TestAssistant_ProcessCommand(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	doneChan := make(chan struct{})
	audioSource := &mockAudioSource{
		commands: [][]byte{
			[]byte("prende luz"),
			[]byte("apaga luz"),
		},
	}

	stt := &mockSTT{
		transcriptions: map[string]string{
			"prende luz": "prende la luz del living",
			"apaga luz":  "apaga la luz del living",
		},
	}

	intentParser := &mockIntentParser{
		intents: map[string]*domain.Command{
			"prende la luz del living": {
				Action:     domain.ActionTurnOn,
				TargetName: "Luz Living",
				TargetType: domain.TargetTypeDevice,
				Confidence: 0.95,
			},
			"apaga la luz del living": {
				Action:     domain.ActionTurnOff,
				TargetName: "Luz Living",
				TargetType: domain.TargetTypeDevice,
				Confidence: 0.95,
			},
		},
	}

	controller := &mockDeviceController{
		doneChan: doneChan,
		expectedCommands: 2,
	}

	registry := &mockRegistry{
		devices: []domain.Device{
			{ID: "dev123", Name: "Luz Living", Type: domain.DeviceTypeLight, Online: true},
		},
	}

	assistant := application.NewAssistant(
		audioSource,
		stt,
		intentParser,
		controller,
		registry,
		&application.NoopNotifier{},
		logger,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = assistant.Run(ctx)
	}()

	// Wait for commands to be processed or timeout
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer timeoutCancel()

	select {
	case <-doneChan:
		// Success
	case <-timeoutCtx.Done():
		t.Fatal("timeout waiting for commands to be processed")
	}

	cancel()

	if len(controller.executedCommands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(controller.executedCommands))
	}
}

func TestAssistant_ProcessScene(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	audioSource := &mockAudioSource{
		commands: [][]byte{[]byte("buenas noches")},
	}

	stt := &mockSTT{
		transcriptions: map[string]string{
			"buenas noches": "activar escena buenas noches",
		},
	}

	intentParser := &mockIntentParser{
		intents: map[string]*domain.Command{
			"activar escena buenas noches": {
				Action:     domain.ActionRunScene,
				TargetName: "Buenas Noches",
				TargetType: domain.TargetTypeScene,
				Confidence: 0.98,
			},
		},
	}

	controller := &mockDeviceController{}

	registry := &mockRegistry{
		scenes: []domain.Scene{
			{ID: "scene456", Name: "Buenas Noches"},
		},
	}

	assistant := application.NewAssistant(
		audioSource,
		stt,
		intentParser,
		controller,
		registry,
		&application.NoopNotifier{},
		logger,
	)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_ = assistant.Run(ctx)
	}()

	cancel()
}

