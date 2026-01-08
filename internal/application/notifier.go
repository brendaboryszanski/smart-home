package application

import "context"

type Notifier interface {
	Notify(ctx context.Context, message string) error
}

type NoopNotifier struct{}

func (n *NoopNotifier) Notify(_ context.Context, _ string) error {
	return nil
}

