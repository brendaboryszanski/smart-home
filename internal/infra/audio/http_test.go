package audio_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"smart-home/internal/infra/audio"
)

func TestHTTPSource_ReceiveAudio(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	source := audio.NewHTTPSource(":0", "", logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := source.Start(ctx); err != nil {
		t.Fatalf("starting source: %v", err)
	}
	defer source.Stop()

	testAudio := []byte("fake audio data for testing")

	go func() {
		time.Sleep(100 * time.Millisecond)
		source.InjectAudio(testAudio)
	}()

	received, err := source.NextCommand(ctx)
	if err != nil {
		t.Fatalf("receiving audio: %v", err)
	}

	if !bytes.Equal(received, testAudio) {
		t.Errorf("audio mismatch: got %d bytes, want %d bytes", len(received), len(testAudio))
	}
}

func TestHTTPSource_HandleAudioEndpoint(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	source := audio.NewHTTPSource(":0", "", logger)

	handler := source.Handler()

	testAudio := []byte("test audio content")
	req := httptest.NewRequest(http.MethodPost, "/audio", bytes.NewReader(testAudio))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("status code: got %d, want %d", rec.Code, http.StatusAccepted)
	}
}

func TestHTTPSource_AlexaEndpointWithToken(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authToken := "test-secret-token-123"
	source := audio.NewHTTPSource(":0", authToken, logger)

	handler := source.Handler()

	tests := []struct {
		name       string
		token      string
		method     string
		wantStatus int
	}{
		{
			name:       "valid token in header",
			token:      authToken,
			method:     "header",
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "valid token in query",
			token:      authToken,
			method:     "query",
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "invalid token",
			token:      "wrong-token",
			method:     "header",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing token",
			token:      "",
			method:     "header",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testText := []byte("turn on the lights")
			var req *http.Request

			if tt.method == "query" {
				req = httptest.NewRequest(http.MethodPost, "/alexa?token="+tt.token, bytes.NewReader(testText))
			} else {
				req = httptest.NewRequest(http.MethodPost, "/alexa", bytes.NewReader(testText))
				if tt.token != "" {
					req.Header.Set("X-Auth-Token", tt.token)
				}
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status code: got %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHTTPSource_AlexaEndpointWithoutToken(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	source := audio.NewHTTPSource(":0", "", logger) // No token configured

	handler := source.Handler()

	testText := []byte("turn on the lights")
	req := httptest.NewRequest(http.MethodPost, "/alexa", bytes.NewReader(testText))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should accept request when no token is configured
	if rec.Code != http.StatusAccepted {
		t.Errorf("status code: got %d, want %d (auth should be disabled)", rec.Code, http.StatusAccepted)
	}
}

func TestFileSource_LoadFromDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		filename string
		content  []byte
	}{
		{"command1.wav", []byte("RIFF....WAVEfmt audio data 1")},
		{"command2.wav", []byte("RIFF....WAVEfmt audio data 2")},
	}

	for _, tc := range testCases {
		path := filepath.Join(tmpDir, tc.filename)
		if err := os.WriteFile(path, tc.content, 0644); err != nil {
			t.Fatalf("writing test file: %v", err)
		}
	}

	source := audio.NewFileSource(tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := source.Start(ctx); err != nil {
		t.Fatalf("starting source: %v", err)
	}

	audio1, err := source.NextCommand(ctx)
	if err != nil {
		t.Fatalf("reading first command: %v", err)
	}

	if len(audio1) == 0 {
		t.Error("first audio is empty")
	}

	audio2, err := source.NextCommand(ctx)
	if err != nil {
		t.Fatalf("reading second command: %v", err)
	}

	if len(audio2) == 0 {
		t.Error("second audio is empty")
	}
}

func TestFileSource_LoadSampleAudios(t *testing.T) {
	samplesDir := "../../../testdata/audio"

	if _, err := os.Stat(samplesDir); os.IsNotExist(err) {
		t.Skip("testdata/audio directory not found, skipping sample audio tests")
	}

	source := audio.NewFileSource(samplesDir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := source.Start(ctx); err != nil {
		t.Fatalf("starting source: %v", err)
	}

	audioData, err := source.NextCommand(ctx)
	if err != nil {
		if ctx.Err() != nil {
			t.Skip("no audio files in testdata/audio")
		}
		t.Fatalf("reading audio: %v", err)
	}

	if len(audioData) < 44 {
		t.Error("audio too short to be valid WAV")
	}

	t.Logf("loaded sample audio: %d bytes", len(audioData))
}

