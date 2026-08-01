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

// Controller owns reporter lifecycle for one invocation.
type Controller interface {
	Reporter() diagnostics.Reporter
	Mode() OutputMode
	Options() ResolvedOptions
	Capabilities() Capabilities
	Suspend(ctx context.Context) error
	Resume(ctx context.Context) error
	Close(ctx context.Context, outcome Outcome) error
	SetRunnerCommand(cmd string)
	SetTerminalIntent(intent TerminalIntent)
	DisableWorkspaceLiveStatus()
	TerminalIntent() TerminalIntent
	ShouldSuspendForChild() bool
}

type controller struct {
	mu        sync.Mutex
	resolved  ResolvedOptions
	caps      Capabilities
	reporter  diagnostics.Reporter
	streams   StreamWriters
	sink      ProgressSink
	runner    *RunnerPresentation
	intent    TerminalIntent
	suspended bool
	closed    bool
	closeErr  error
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

	settings := Effective(resolved, caps)
	settings.BinaryName = resolved.BinaryName
	if !resolved.Structured() {
		mapOpts := MapOptions{Debug: resolved.Debug, Redact: diagnostics.Redact, BinaryName: resolved.BinaryName}
		opts.HumanErrorRender = func(err error) string {
			return NewStaticRenderer(settings).Error(MapError(err, mapOpts))
		}
	}

	sink, err := selectProgressSink(resolved, caps, settings, streams.Err)
	if err != nil {
		return nil, err
	}
	attachProgressHooks(&opts, sink)

	var runnerPres *RunnerPresentation
	if !resolved.Structured() && resolved.Output != OutputSilent {
		runnerPres = newRunnerPresentation(settings, streams.Err, resolved.Debug, resolved.Summary)
		attachRunnerHooks(&opts, runnerPres)
	}

	c := &controller{
		resolved: resolved,
		caps:     caps,
		reporter: diagnostics.NewReporter(opts),
		streams:  streams,
		sink:     sink,
		runner:   runnerPres,
		intent:   TerminalAuto,
	}
	return c, nil
}

// selectProgressSink picks live, static-rich, plain, or nil progress rendering for human modes.
func selectProgressSink(resolved ResolvedOptions, caps Capabilities, settings EffectiveSettings, errW io.Writer) (ProgressSink, error) {
	if resolved.Structured() || resolved.Output == OutputSilent {
		return nil, nil
	}
	if !resolved.Progress {
		return nil, nil
	}

	// Rich output: use activity renderer on TTY, static-rich otherwise.
	if resolved.Output == OutputRich {
		if caps.StderrTTY {
			return NewActivityProgressRenderer(errW, settings), nil
		}
		return NewStaticRichProgressRenderer(errW, settings), nil
	}

	// Plain output: use plain renderer.
	return NewPlainProgressRenderer(errW), nil
}

func (c *controller) Reporter() diagnostics.Reporter { return c.reporter }

func (c *controller) Mode() OutputMode { return c.resolved.Output }

func (c *controller) Options() ResolvedOptions { return c.resolved }

func (c *controller) Capabilities() Capabilities { return c.caps }

func (c *controller) SetRunnerCommand(cmd string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runner != nil {
		c.runner.SetCommand(cmd)
	}
}

func (c *controller) SetTerminalIntent(intent TerminalIntent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.intent = intent
	if c.runner != nil {
		c.runner.SetIntent(intent)
	}
}

func (c *controller) TerminalIntent() TerminalIntent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.intent
}

func (c *controller) ShouldSuspendForChild() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return ShouldSuspendRichUI(c.intent, c.caps)
}

func (c *controller) DisableWorkspaceLiveStatus() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runner != nil {
		c.runner.DisableWorkspaceLiveStatus()
	}
}

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
	if c.sink != nil {
		c.sink.Suspend()
	}
	if c.runner != nil {
		c.runner.Suspend()
	}
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
	if c.sink != nil {
		c.sink.Resume()
	}
	if c.runner != nil {
		c.runner.Resume()
	}
	return nil
}

func (c *controller) Close(ctx context.Context, outcome Outcome) error {
	c.mu.Lock()
	if c.closed {
		err := c.closeErr
		c.mu.Unlock()
		return err
	}
	c.closed = true
	sink := c.sink
	c.sink = nil
	c.mu.Unlock()

	cleanupCtx, cancel := cleanupContext(ctx)
	defer cancel()
	_ = cleanupCtx
	_ = outcome

	var closeErr error
	if sink != nil {
		closeErr = sink.Close()
	}
	c.mu.Lock()
	c.closeErr = closeErr
	c.mu.Unlock()
	return closeErr
}

func cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.WithTimeout(context.Background(), 2*time.Second)
	}
	return context.WithTimeout(ctx, 2*time.Second)
}
