package tuya

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"smart-home/internal/domain"
)

type Registry struct {
	client *Client
	logger *slog.Logger

	mu      sync.RWMutex
	devices []domain.Device
	scenes  []domain.Scene

	deviceIndex map[string]*domain.Device
	sceneIndex  map[string]*domain.Scene
}

func NewRegistry(client *Client, logger *slog.Logger) *Registry {
	return &Registry{
		client:      client,
		logger:      logger,
		deviceIndex: make(map[string]*domain.Device),
		sceneIndex:  make(map[string]*domain.Scene),
	}
}

func (r *Registry) Sync(ctx context.Context) error {
	r.logger.Info("syncing devices and scenes from Tuya")

	devices, err := r.client.GetDevices(ctx)
	if err != nil {
		return fmt.Errorf("fetching devices: %w", err)
	}

	homes, err := r.client.GetHomes(ctx)
	if err != nil {
		return fmt.Errorf("fetching homes: %w", err)
	}

	var scenes []domain.Scene
	for _, homeID := range homes {
		homeScenes, err := r.client.GetScenes(ctx, homeID)
		if err != nil {
			r.logger.Warn("failed to fetch scenes for home", "homeID", homeID, "error", err)
			continue
		}
		scenes = append(scenes, homeScenes...)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.devices = devices
	r.scenes = scenes

	r.deviceIndex = make(map[string]*domain.Device)
	for i := range r.devices {
		key := strings.ToLower(r.devices[i].Name)
		r.deviceIndex[key] = &r.devices[i]
	}

	r.sceneIndex = make(map[string]*domain.Scene)
	for i := range r.scenes {
		key := strings.ToLower(r.scenes[i].Name)
		r.sceneIndex[key] = &r.scenes[i]
	}

	r.logger.Info("sync complete",
		"devices", len(r.devices),
		"scenes", len(r.scenes),
	)

	return nil
}

func (r *Registry) GetDevices() []domain.Device {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.Device, len(r.devices))
	copy(result, r.devices)
	return result
}

func (r *Registry) GetScenes() []domain.Scene {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.Scene, len(r.scenes))
	copy(result, r.scenes)
	return result
}

func (r *Registry) FindDeviceByName(name string) (*domain.Device, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := strings.ToLower(strings.TrimSpace(name))

	if d, ok := r.deviceIndex[key]; ok {
		return d, true
	}

	for _, d := range r.devices {
		if strings.Contains(strings.ToLower(d.Name), key) {
			return &d, true
		}
	}

	return nil, false
}

func (r *Registry) FindSceneByName(name string) (*domain.Scene, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := strings.ToLower(strings.TrimSpace(name))

	if s, ok := r.sceneIndex[key]; ok {
		return s, true
	}

	for _, s := range r.scenes {
		if strings.Contains(strings.ToLower(s.Name), key) {
			return &s, true
		}
	}

	return nil, false
}

func (r *Registry) Summary() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString("## Dispositivos disponibles:\n")
	for _, d := range r.devices {
		status := "offline"
		if d.Online {
			status = "online"
		}
		sb.WriteString(fmt.Sprintf("- %s (tipo: %s, estado: %s)\n", d.Name, d.Type, status))
	}

	sb.WriteString("\n## Escenas disponibles:\n")
	for _, s := range r.scenes {
		sb.WriteString(fmt.Sprintf("- %s\n", s.Name))
	}

	return sb.String()
}

func (r *Registry) StartPeriodicSync(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.Sync(ctx); err != nil {
					r.logger.Error("periodic sync failed", "error", err)
				}
			}
		}
	}()
}

