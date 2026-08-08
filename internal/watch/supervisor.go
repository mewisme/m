package watch

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// DefaultDebounceInterval is the quiet period before a restart after file changes.
const DefaultDebounceInterval = 200 * time.Millisecond

// RestartFunc starts a child process and blocks until it exits.
// ctx is cancelled to request graceful termination.
type RestartFunc func(ctx context.Context) (int, error)

// SupervisorOptions configures the watch-restart loop.
type SupervisorOptions struct {
	Watcher          Watcher
	WatchPaths       []string // used when Graph is nil (backward compat)
	Restart          RestartFunc
	ClearScreen      bool
	DebounceInterval time.Duration
	OnRestart        func(reason string)

	// Graph is the logical dependency graph. When non-nil the supervisor
	// registers Graph.WatchPaths(), filters events through
	// Graph.ShouldTrigger, and reconciles coverage after each child
	// exit via ReconcilePaths.
	Graph *DependencyGraph
	// ReconcilePaths returns canonical paths to add/remove after a
	// child exits. code is the child's exit code; return nil,nil to
	// keep current coverage (e.g. preserve dependencies on failure).
	ReconcilePaths func(code int) (add, remove []string)
}

// Supervisor runs the watch-restart loop.
type Supervisor struct {
	opts SupervisorOptions
}

// NewSupervisor creates a new supervisor.
func NewSupervisor(opts SupervisorOptions) *Supervisor {
	if opts.DebounceInterval <= 0 {
		opts.DebounceInterval = DefaultDebounceInterval
	}
	return &Supervisor{opts: opts}
}

// Run starts the watch-restart loop. Blocks until ctx is cancelled or
// an unrecoverable error occurs.
func (s *Supervisor) Run(ctx context.Context) (int, error) {
	w := s.opts.Watcher
	if w == nil {
		return 1, fmt.Errorf("watch: nil watcher")
	}

	g := s.opts.Graph
	if g != nil {
		for _, p := range g.WatchPaths() {
			if err := w.Add(p); err != nil {
				fmt.Fprintf(os.Stderr, "watch: cannot watch %s: %v\n", p, err)
			}
		}
	} else {
		for _, p := range s.opts.WatchPaths {
			if err := w.Add(p); err != nil {
				fmt.Fprintf(os.Stderr, "watch: cannot watch %s: %v\n", p, err)
			}
		}
	}

	// Forward watcher errors to stderr.
	go func() {
		for err := range w.Errors() {
			if err != nil {
				fmt.Fprintf(os.Stderr, "watch: %v\n", err)
			}
		}
	}()

	eventCh := w.Events()
	debounce := s.opts.DebounceInterval

	var mu sync.Mutex
	var debounceTimer *time.Timer
	triggerRestart := make(chan struct{}, 1)

	notifyChange := func() {
		select {
		case triggerRestart <- struct{}{}:
		default:
		}
	}

	// Consume events in background, feeding debounce.
	// When a graph is active, filter events through ShouldTrigger.
	go func() {
		for evt := range eventCh {
			if g != nil && !g.ShouldTrigger(evt.Path) {
				continue
			}
			mu.Lock()
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounce, notifyChange)
			mu.Unlock()
		}
	}()

	lastCode := 0
	for {
		if s.opts.ClearScreen {
			fmt.Fprint(os.Stderr, "\033[2J\033[H")
		}
		if s.opts.OnRestart != nil {
			s.opts.OnRestart("starting")
		}

		childCtx, cancelChild := context.WithCancel(ctx)

		childDone := make(chan struct {
			code int
			err  error
		}, 1)
		go func() {
			code, err := s.opts.Restart(childCtx)
			childDone <- struct {
				code int
				err  error
			}{code, err}
		}()

		select {
		case <-triggerRestart:
			if s.opts.OnRestart != nil {
				s.opts.OnRestart("file changed")
			}
			cancelChild()
			result := <-childDone
			lastCode = result.code
			if result.err != nil && result.err != context.Canceled {
				fmt.Fprintf(os.Stderr, "watch: %v\n", result.err)
			}
			s.reconcile(g, lastCode, w)

		case result := <-childDone:
			cancelChild()
			lastCode = result.code
			if result.err != nil && result.err != context.Canceled {
				fmt.Fprintf(os.Stderr, "watch: %v\n", result.err)
			}
			s.reconcile(g, lastCode, w)
			if s.opts.OnRestart != nil {
				s.opts.OnRestart(fmt.Sprintf("child exited (code %d)", result.code))
			}
			select {
			case <-triggerRestart:
			case <-ctx.Done():
				return lastCode, ctx.Err()
			}

		case <-ctx.Done():
			cancelChild()
			<-childDone
			return lastCode, ctx.Err()
		}

		// Drain any queued restart triggers before next iteration.
		select {
		case <-triggerRestart:
		default:
		}
	}
}

// reconcile applies graph coverage updates after a child exits.
func (s *Supervisor) reconcile(g *DependencyGraph, code int, w Watcher) {
	if g == nil || s.opts.ReconcilePaths == nil {
		return
	}
	add, remove := s.opts.ReconcilePaths(code)
	for _, p := range remove {
		if err := w.Remove(p); err != nil {
			fmt.Fprintf(os.Stderr, "watch: cannot unwatch %s: %v\n", p, err)
		}
	}
	for _, p := range add {
		if err := w.Add(p); err != nil {
			fmt.Fprintf(os.Stderr, "watch: cannot watch %s: %v\n", p, err)
		}
	}
}