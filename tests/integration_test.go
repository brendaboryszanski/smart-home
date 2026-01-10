package tests

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"smart-home/internal/application"
	"smart-home/internal/domain"
	"smart-home/internal/infra/audio"
)

type testRecorder struct {
	transcriptions []string
	commands       []*domain.Command
	executedCmds   []*domain.Command
	triggeredScns  []string
}

type recordingSTT struct {
	recorder *testRecorder
	results  map[int]string
	callNum  int
}

func (r *recordingSTT) Transcribe(_ context.Context, _ []byte) (string, error) {
	text := "comando desconocido"
	if t, ok := r.results[r.callNum]; ok {
		text = t
	}
	r.recorder.transcriptions = append(r.recorder.transcriptions, text)
	r.callNum++
	return text, nil
}

type recordingIntent struct {
	recorder *testRecorder
	results  map[string]*domain.Command
}

func (r *recordingIntent) Parse(_ context.Context, text string, _ application.DeviceRegistry) (*domain.Command, error) {
	cmd := &domain.Command{Action: domain.ActionUnknown}
	if c, ok := r.results[text]; ok {
		cmd = c
	}
	r.recorder.commands = append(r.recorder.commands, cmd)
	return cmd, nil
}

type recordingController struct {
	recorder *testRecorder
}

func (r *recordingController) ExecuteCommand(_ context.Context, cmd *domain.Command) error {
	r.recorder.executedCmds = append(r.recorder.executedCmds, cmd)
	return nil
}

func (r *recordingController) TriggerScene(_ context.Context, sceneID string) error {
	r.recorder.triggeredScns = append(r.recorder.triggeredScns, sceneID)
	return nil
}

type staticRegistry struct {
	devices []domain.Device
	scenes  []domain.Scene
}

func (s *staticRegistry) Sync(_ context.Context) error          { return nil }
func (s *staticRegistry) GetDevices() []domain.Device           { return s.devices }
func (s *staticRegistry) GetScenes() []domain.Scene             { return s.scenes }
func (s *staticRegistry) Summary() string                       { return "test devices" }
func (s *staticRegistry) FindDeviceByName(name string) (*domain.Device, bool) {
	for i, d := range s.devices {
		if d.Name == name {
			return &s.devices[i], true
		}
	}
	return nil, false
}
func (s *staticRegistry) FindSceneByName(name string) (*domain.Scene, bool) {
	for i, sc := range s.scenes {
		if sc.Name == name {
			return &s.scenes[i], true
		}
	}
	return nil, false
}

func (s *staticRegistry) StartPeriodicSync(_ context.Context, _ time.Duration) {}

func TestIntegration_WithSampleAudios(t *testing.T) {
	audioDir := "../testdata/audio"

	files, err := filepath.Glob(filepath.Join(audioDir, "*.wav"))
	if err != nil || len(files) == 0 {
		t.Skip("No sample audio files found in testdata/audio")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recorder := &testRecorder{}

	audioSource := audio.NewFileSource(audioDir)

	stt := &recordingSTT{
		recorder: recorder,
		results: map[int]string{
			0: "prende la luz",
			1: "apaga todo",
		},
	}

	intent := &recordingIntent{
		recorder: recorder,
		results: map[string]*domain.Command{
			"prende la luz": {
				Action:     domain.ActionTurnOn,
				TargetName: "Luz Living",
				TargetType: domain.TargetTypeDevice,
			},
			"apaga todo": {
				Action:     domain.ActionRunScene,
				TargetName: "Apagar Todo",
				TargetType: domain.TargetTypeScene,
			},
		},
	}

	controller := &recordingController{recorder: recorder}

	registry := &staticRegistry{
		devices: []domain.Device{
			{ID: "d1", Name: "Luz Living", Type: domain.DeviceTypeLight},
		},
		scenes: []domain.Scene{
			{ID: "s1", Name: "Apagar Todo"},
		},
	}

	assistant := application.NewAssistant(
		audioSource,
		stt,
		intent,
		controller,
		registry,
		&application.NoopNotifier{},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = assistant.Run(ctx)
	}()

	<-ctx.Done()

	t.Logf("Transcriptions: %v", recorder.transcriptions)
	t.Logf("Commands parsed: %d", len(recorder.commands))
	t.Logf("Commands executed: %d", len(recorder.executedCmds))
	t.Logf("Scenes triggered: %v", recorder.triggeredScns)
}

func TestIntegration_TextCommandFromAlexa(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recorder := &testRecorder{}

	httpSource := audio.NewHTTPSource(":0", "", logger)

	stt := &recordingSTT{recorder: recorder, results: map[int]string{}}
	intent := &recordingIntent{
		recorder: recorder,
		results: map[string]*domain.Command{
			"prende cocina": {
				Action:     domain.ActionTurnOn,
				TargetName: "Luz Cocina",
				TargetType: domain.TargetTypeDevice,
			},
		},
	}
	controller := &recordingController{recorder: recorder}
	registry := &staticRegistry{
		devices: []domain.Device{
			{ID: "d2", Name: "Luz Cocina", Type: domain.DeviceTypeLight},
		},
	}

	assistant := application.NewAssistant(
		httpSource,
		stt,
		intent,
		controller,
		registry,
		&application.NoopNotifier{},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_ = assistant.Run(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	httpSource.InjectAudio([]byte("__TEXT__:prende cocina"))

	time.Sleep(500 * time.Millisecond)
	cancel()

	if len(recorder.transcriptions) > 0 {
		t.Error("STT should not be called for text commands")
	}

	if len(recorder.commands) == 0 {
		t.Error("command should have been parsed")
	}
}

func loadTestAudio(path string) ([]byte, error) {
	return os.ReadFile(path)
}

