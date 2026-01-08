package application

import (
	"context"

	"smart-home/internal/domain"
)

type IntentParser interface {
	Parse(ctx context.Context, text string, registry DeviceRegistry) (*domain.Command, error)
}

