package event

import (
	"context"

	"github.com/go-fund-transfer/internal/core"
)

type EventNotifier interface {
	Producer(ctx context.Context, event core.Event) error
}