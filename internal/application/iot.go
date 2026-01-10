package application

import (
	"context"
	"time"

	"smart-home/internal/domain"
)

type DeviceController interface {
	ExecuteCommand(ctx context.Context, cmd *domain.Command) error
	TriggerScene(ctx context.Context, sceneID string) error
}

type DeviceRegistry interface {
	Sync(ctx context.Context) error
	GetDevices() []domain.Device
	GetScenes() []domain.Scene
	FindDeviceByName(name string) (*domain.Device, bool)
	FindSceneByName(name string) (*domain.Scene, bool)
	Summary() string
	StartPeriodicSync(ctx context.Context, interval time.Duration)
}

