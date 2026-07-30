package presentation

import (
	"context"
	"time"
)

// Clock abstracts time for tests.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// DefaultClock is the production clock.
var DefaultClock Clock = systemClock{}

// CleanupTimeout is the upper bound for presentation teardown.
const CleanupTimeout = 2 * time.Second

// WithCleanupTimeout derives a bounded context from parent ctx.
func WithCleanupTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.WithTimeout(context.Background(), CleanupTimeout)
	}
	return context.WithTimeout(ctx, CleanupTimeout)
}
