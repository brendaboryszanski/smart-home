package application

import "context"

type SpeechToText interface {
	Transcribe(ctx context.Context, audio []byte) (string, error)
}

