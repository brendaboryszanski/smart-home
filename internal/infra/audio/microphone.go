//go:build portaudio
// +build portaudio

package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
)

type MicrophoneSource struct {
	stream     *portaudio.Stream
	wakeWord   string
	sampleRate int
	logger     *slog.Logger

	mu        sync.Mutex
	buffer    []int16
	recording bool
}

func NewMicrophoneSource(wakeWord string, sampleRate int, logger *slog.Logger) *MicrophoneSource {
	return &MicrophoneSource{
		wakeWord:   wakeWord,
		sampleRate: sampleRate,
		logger:     logger,
		buffer:     make([]int16, 0),
	}
}

func (m *MicrophoneSource) Name() string {
	return "microphone"
}

func (m *MicrophoneSource) Start(_ context.Context) error {
	if err := portaudio.Initialize(); err != nil {
		return fmt.Errorf("initializing portaudio: %w", err)
	}

	inputChannels := 1
	outputChannels := 0
	framesPerBuffer := 1024

	buffer := make([]int16, framesPerBuffer)

	stream, err := portaudio.OpenDefaultStream(
		inputChannels,
		outputChannels,
		float64(m.sampleRate),
		framesPerBuffer,
		buffer,
	)
	if err != nil {
		return fmt.Errorf("opening stream: %w", err)
	}

	m.stream = stream

	if err := m.stream.Start(); err != nil {
		return fmt.Errorf("starting stream: %w", err)
	}

	m.logger.Info("microphone started", "sampleRate", m.sampleRate)
	return nil
}

func (m *MicrophoneSource) Stop() error {
	if m.stream != nil {
		m.stream.Stop()
		m.stream.Close()
	}
	portaudio.Terminate()
	return nil
}

func (m *MicrophoneSource) NextCommand(ctx context.Context) ([]byte, error) {
	m.logger.Info("waiting for wake word", "wakeWord", m.wakeWord)

	samples := make([]int16, 0, m.sampleRate*5)
	silenceThreshold := int16(500)
	silenceDuration := 0
	maxSilenceFrames := m.sampleRate

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		buffer := make([]int16, 1024)
		if err := m.stream.Read(); err != nil {
			return nil, fmt.Errorf("reading from stream: %w", err)
		}

		samples = append(samples, buffer...)

		isSilent := true
		for _, sample := range buffer {
			if sample > silenceThreshold || sample < -silenceThreshold {
				isSilent = false
				break
			}
		}

		if isSilent {
			silenceDuration += len(buffer)
		} else {
			silenceDuration = 0
		}

		if silenceDuration > maxSilenceFrames && len(samples) > m.sampleRate {
			break
		}

		if len(samples) > m.sampleRate*10 {
			break
		}
	}

	return samplesToWav(samples, m.sampleRate)
}

func samplesToWav(samples []int16, sampleRate int) ([]byte, error) {
	var buf bytes.Buffer

	dataSize := len(samples) * 2
	fileSize := 36 + dataSize

	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, int32(fileSize))
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, int32(16))
	binary.Write(&buf, binary.LittleEndian, int16(1))
	binary.Write(&buf, binary.LittleEndian, int16(1))
	binary.Write(&buf, binary.LittleEndian, int32(sampleRate))
	binary.Write(&buf, binary.LittleEndian, int32(sampleRate*2))
	binary.Write(&buf, binary.LittleEndian, int16(2))
	binary.Write(&buf, binary.LittleEndian, int16(16))

	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, int32(dataSize))
	for _, sample := range samples {
		binary.Write(&buf, binary.LittleEndian, sample)
	}

	return buf.Bytes(), nil
}

type WakeWordDetector struct {
	wakeWords []string
	timeout   time.Duration
}

func NewWakeWordDetector(wakeWords []string, timeout time.Duration) *WakeWordDetector {
	return &WakeWordDetector{
		wakeWords: wakeWords,
		timeout:   timeout,
	}
}

