package application

import "context"

type AudioSource interface {
	Start(ctx context.Context) error
	Stop() error
	NextCommand(ctx context.Context) ([]byte, error)
	Name() string
}

type AudioFormat struct {
	SampleRate int
	Channels   int
	BitDepth   int
}

func DefaultAudioFormat() AudioFormat {
	return AudioFormat{
		SampleRate: 16000,
		Channels:   1,
		BitDepth:   16,
	}
}

