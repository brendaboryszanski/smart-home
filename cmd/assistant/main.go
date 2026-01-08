package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"smart-home/config"
	"smart-home/internal/application"
	"smart-home/internal/infra/anthropic"
	"smart-home/internal/infra/audio"
	"smart-home/internal/infra/openai"
	"smart-home/internal/infra/pushover"
	"smart-home/internal/infra/tuya"
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

	whisperClient := openai.NewWhisperClient(cfg.OpenAI.APIKey, cfg.OpenAI.Language)
	claudeClient := anthropic.NewClaudeClient(cfg.Anthropic.APIKey, cfg.Anthropic.Model)

	tuyaClient := tuya.NewClient(cfg.Tuya.ClientID, cfg.Tuya.Secret, cfg.Tuya.Region)
	registry := tuya.NewRegistry(tuyaClient, logger)

	syncInterval, err := time.ParseDuration(cfg.Tuya.SyncInterval)
	if err != nil {
		logger.Warn("invalid sync interval, using default", "error", err, "value", cfg.Tuya.SyncInterval)
		syncInterval = 5 * time.Minute
	}
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
		whisperClient,
		claudeClient,
		tuyaClient,
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

