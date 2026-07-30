package presentation

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/mewisme/mew/internal/diagnostics"
)

// StreamWriters binds stdout/stderr for one invocation.
type StreamWriters struct {
	Out io.Writer
	Err io.Writer
}

// Controller owns reporter lifecycle for one CLI invocation.
type Controller interface {
	Reporter() diagnostics.Reporter
	Mode() OutputMode
	EffectiveMode() OutputMode
	Options() ResolvedOptions
	Capabilities() Capabilities
	Suspend(ctx context.Context) error
	Resume(ctx context.Context) error
	Close(ctx context.Context, outcome Outcome) error
}

type controller struct {
	mu          sync.Mutex
	resolved    ResolvedOptions
	caps        Capabilities
	reporter    diagnostics.Reporter
	streams     StreamWriters
	suspended   bool
	closed      bool
	closeErr    error
	debugOnDown bool
}

// NewController builds a presentation controller and diagnostics reporter bridge.
func NewController(resolved ResolvedOptions, caps Capabilities, streams StreamWriters) (Controller, error) {
	if streams.Out == nil {
		streams.Out = io.Discard
	}
	if streams.Err == nil {
		streams.Err = io.Discard
	}
	opts := diagnostics.Options{
		Format:    resolved.ReporterFormat(),
		Debug:     resolved.Debug,
		Color:     resolved.ColorMode(),
		Unsafe:    resolved.Unsafe,
		Out:       streams.Out,
		Err:       streams.Err,
		TermWidth: resolved.TermWidth,
	}
	if !resolved.Legacy && !resolved.Structured() {
		settings := Effective(resolved, caps)
		mapOpts := MapOptions{Debug: resolved.Debug, Redact: diagnostics.Redact}
		opts.HumanErrorRender = func(err error) string {
			return NewStaticRenderer(settings).Error(MapError(err, mapOpts))
		}
	}
	c := &controller{
		resolved:    resolved,
		caps:        caps,
		reporter:    diagnostics.NewReporter(opts),
		streams:     streams,
		debugOnDown: resolved.DowngradedRich,
	}
	if c.debugOnDown {
		c.reporter.Debug("presentation: rich mode downgraded to plain")
	}
	return c, nil
}

func (c *controller) Reporter() diagnostics.Reporter { return c.reporter }

func (c *controller) Mode() OutputMode { return c.resolved.RequestedOutput }

func (c *controller) EffectiveMode() OutputMode { return c.resolved.EffectiveOutput }

func (c *controller) Options() ResolvedOptions { return c.resolved }

func (c *controller) Capabilities() Capabilities { return c.caps }

func (c *controller) Suspend(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.suspended = true
	return nil
}

func (c *controller) Resume(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.suspended = false
	return nil
}

func (c *controller) Close(ctx context.Context, outcome Outcome) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return c.closeErr
	}
	c.closed = true
	cleanupCtx, cancel := cleanupContext(ctx)
	defer cancel()
	_ = cleanupCtx
	_ = outcome
	// ponytail: live renderer teardown lands in UX-0002/0004; Close stays idempotent now.
	c.closeErr = nil
	return c.closeErr
}

// cleanupContext returns a bounded context for renderer teardown.
func cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.WithTimeout(context.Background(), 2*time.Second)
	}
	return context.WithTimeout(ctx, 2*time.Second)
}
