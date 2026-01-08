package application

import (
	"context"
	"fmt"
	"log/slog"

	"smart-home/internal/domain"
)

type Assistant struct {
	audio    AudioSource
	stt      SpeechToText
	intent   IntentParser
	iot      DeviceController
	registry DeviceRegistry
	notifier Notifier
	logger   *slog.Logger
}

func NewAssistant(
	audio AudioSource,
	stt SpeechToText,
	intent IntentParser,
	iot DeviceController,
	registry DeviceRegistry,
	notifier Notifier,
	logger *slog.Logger,
) *Assistant {
	return &Assistant{
		audio:    audio,
		stt:      stt,
		intent:   intent,
		iot:      iot,
		registry: registry,
		notifier: notifier,
		logger:   logger,
	}
}

func (a *Assistant) Run(ctx context.Context) error {
	a.logger.Info("syncing device registry")
	if err := a.registry.Sync(ctx); err != nil {
		return fmt.Errorf("initial registry sync: %w", err)
	}

	a.logger.Info("starting audio source", "source", a.audio.Name())
	if err := a.audio.Start(ctx); err != nil {
		return fmt.Errorf("starting audio: %w", err)
	}
	defer a.audio.Stop()

	a.logger.Info("assistant ready, listening for commands")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := a.processOneCommand(ctx); err != nil {
				a.logger.Error("processing command", "error", err)
			}
		}
	}
}

func (a *Assistant) processOneCommand(ctx context.Context) error {
	audioData, err := a.audio.NextCommand(ctx)
	if err != nil {
		return fmt.Errorf("getting audio: %w", err)
	}

	if len(audioData) == 0 {
		return nil
	}

	var text string

	if directText, isText := isTextCommand(audioData); isText {
		a.logger.Info("received text command directly", "text", directText)
		text = directText
	} else {
		a.logger.Info("received audio", "bytes", len(audioData))

		var err error
		text, err = a.stt.Transcribe(ctx, audioData)
		if err != nil {
			return fmt.Errorf("transcribing: %w", err)
		}

		a.logger.Info("transcribed", "text", text)
	}

	cmd, err := a.intent.Parse(ctx, text, a.registry)
	if err != nil {
		return fmt.Errorf("parsing intent: %w", err)
	}

	a.logger.Info("parsed intent",
		"action", cmd.Action,
		"target", cmd.TargetName,
		"confidence", cmd.Confidence,
	)

	if cmd.Action == domain.ActionUnknown {
		a.logger.Warn("unknown command, skipping", "text", text)
		return nil
	}

	result, err := a.executeCommand(ctx, cmd)
	if err != nil {
		notifyErr := a.notifier.Notify(ctx, fmt.Sprintf("Error: %s", err.Error()))
		if notifyErr != nil {
			a.logger.Error("notifying error", "error", notifyErr)
		}
		return fmt.Errorf("executing: %w", err)
	}

	if err := a.notifier.Notify(ctx, result); err != nil {
		a.logger.Error("notifying result", "error", err)
	}

	return nil
}

func isTextCommand(data []byte) (string, bool) {
	if len(data) > len(domain.TextCommandPrefix) && string(data[:len(domain.TextCommandPrefix)]) == domain.TextCommandPrefix {
		return string(data[len(domain.TextCommandPrefix):]), true
	}
	return "", false
}

func (a *Assistant) executeCommand(ctx context.Context, cmd *domain.Command) (string, error) {
	switch cmd.TargetType {
	case domain.TargetTypeScene:
		scene, ok := a.registry.FindSceneByName(cmd.TargetName)
		if !ok {
			return "", fmt.Errorf("scene not found: %s", cmd.TargetName)
		}
		if err := a.iot.TriggerScene(ctx, scene.ID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Scene '%s' executed", cmd.TargetName), nil

	case domain.TargetTypeDevice:
		device, ok := a.registry.FindDeviceByName(cmd.TargetName)
		if !ok {
			return "", fmt.Errorf("device not found: %s", cmd.TargetName)
		}
		cmd.TargetID = device.ID
		if err := a.iot.ExecuteCommand(ctx, cmd); err != nil {
			return "", err
		}
		return fmt.Sprintf("Command '%s' executed on '%s'", cmd.Action, cmd.TargetName), nil

	default:
		return "", fmt.Errorf("unknown target type: %s", cmd.TargetType)
	}
}

