package application

import (
	"context"
	"fmt"
)

type SpeechToText interface {
	Transcribe(ctx context.Context, audio []byte) (string, error)
}

// NoopSTT is a no-op speech-to-text client for text-only sources (e.g., Alexa).
// It returns an error if called with actual audio data.
type NoopSTT struct{}

func (n *NoopSTT) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "", fmt.Errorf("speech-to-text not configured: set openai.api_key to enable audio transcription")
}

