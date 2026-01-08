//go:build !portaudio
// +build !portaudio

package audio

import (
	"context"
	"fmt"
	"log/slog"
)

// MicrophoneSource stub when portaudio is not available
type MicrophoneSource struct {
	logger *slog.Logger
}

func NewMicrophoneSource(wakeWord string, sampleRate int, logger *slog.Logger) *MicrophoneSource {
	return &MicrophoneSource{logger: logger}
}

func (m *MicrophoneSource) Name() string {
	return "microphone"
}

func (m *MicrophoneSource) Start(_ context.Context) error {
	return fmt.Errorf("microphone source not available: rebuild with -tags portaudio")
}

func (m *MicrophoneSource) Stop() error {
	return nil
}

func (m *MicrophoneSource) NextCommand(_ context.Context) ([]byte, error) {
	return nil, fmt.Errorf("microphone source not available")
}
