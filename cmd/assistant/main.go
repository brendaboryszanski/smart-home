package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"smart-home/config"
	"smart-home/internal/application"
	"smart-home/internal/infra/anthropic"
	"smart-home/internal/infra/audio"
	"smart-home/internal/infra/gemini"
	"smart-home/internal/infra/homeassistant"
	"smart-home/internal/infra/openai"
	"smart-home/internal/infra/pushover"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("loading config", "error", err)
		os.Exit(1)
	}

	logger := setupLogger(cfg.Log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("shutting down")
		cancel()
	}()

	audioSource := createAudioSource(cfg.Audio, logger)

	// Create STT client only if needed (not needed for text-only sources like Alexa)
	sttClient := createSTTClient(cfg, logger)

	// Create intent parser (Anthropic or Gemini - whichever has an API key)
	intentParser, err := createIntentParser(cfg, logger)
	if err != nil {
		logger.Error("creating intent parser", "error", err)
		os.Exit(1)
	}

	// Create IoT controller and registry (Home Assistant or Tuya)
	iotController, registry, syncInterval := createIoTBackend(cfg, logger)
	if syncInterval > 0 {
		registry.StartPeriodicSync(ctx, syncInterval)
	}

	var notifier application.Notifier
	if cfg.Pushover.Enabled {
		notifier = pushover.NewClient(cfg.Pushover.Token, cfg.Pushover.UserKey)
	} else {
		notifier = &application.NoopNotifier{}
	}

	assistant := application.NewAssistant(
		audioSource,
		sttClient,
		intentParser,
		iotController,
		registry,
		notifier,
		logger,
	)

	logger.Info("starting smart home assistant",
		"audio_source", cfg.Audio.Source,
	)

	if err := assistant.Run(ctx); err != nil && err != context.Canceled {
		logger.Error("assistant error", "error", err)
		os.Exit(1)
	}
}

func createAudioSource(cfg config.AudioConfig, logger *slog.Logger) application.AudioSource {
	switch cfg.Source {
	case "http":
		return audio.NewHTTPSource(cfg.HTTPAddr, cfg.AuthToken, logger)
	case "file":
		return audio.NewFileSource(cfg.FileDir)
	case "microphone":
		return audio.NewMicrophoneSource(cfg.WakeWord, cfg.SampleRate, logger)
	default:
		logger.Warn("unknown audio source, using http", "source", cfg.Source)
		return audio.NewHTTPSource(cfg.HTTPAddr, cfg.AuthToken, logger)
	}
}

func createSTTClient(cfg *config.Config, logger *slog.Logger) application.SpeechToText {
	if cfg.OpenAI.APIKey == "" {
		logger.Info("no OpenAI API key configured, using noop STT (text commands only)")
		return &application.NoopSTT{}
	}
	logger.Info("using OpenAI Whisper for speech-to-text")
	return openai.NewWhisperClient(cfg.OpenAI.APIKey, cfg.OpenAI.Language)
}

func createIntentParser(cfg *config.Config, logger *slog.Logger) (application.IntentParser, error) {
	// Prefer Anthropic if configured, otherwise use Gemini
	if cfg.Anthropic.APIKey != "" {
		logger.Info("using Anthropic Claude for intent parsing", "model", cfg.Anthropic.Model)
		return anthropic.NewClaudeClient(cfg.Anthropic.APIKey, cfg.Anthropic.Model), nil
	}
	if cfg.Gemini.APIKey != "" {
		logger.Info("using Google Gemini for intent parsing", "model", cfg.Gemini.Model)
		return gemini.NewClient(cfg.Gemini.APIKey, cfg.Gemini.Model), nil
	}
	return nil, fmt.Errorf("no LLM API key configured: set either anthropic.api_key or gemini.api_key")
}

func createIoTBackend(cfg *config.Config, logger *slog.Logger) (application.DeviceController, application.DeviceRegistry, time.Duration) {
	logger.Info("using Home Assistant for device control", "url", cfg.HomeAssistant.URL)
	haClient := homeassistant.NewClient(cfg.HomeAssistant.URL, cfg.HomeAssistant.Token)
	registry := homeassistant.NewRegistry(haClient, logger)

	syncInterval, err := time.ParseDuration(cfg.HomeAssistant.SyncInterval)
	if err != nil {
		logger.Warn("invalid sync interval, using default", "error", err)
		syncInterval = 5 * time.Minute
	}

	return haClient, registry, syncInterval
}

func setupLogger(cfg config.LogConfig) *slog.Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

