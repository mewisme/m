package watch

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakeWatcher implements Watcher for testing the supervisor.
type fakeWatcher struct {
	mu     sync.Mutex
	events chan Event
	errs   chan error
	closed bool
}

func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{
		events: make(chan Event, 64),
		errs:   make(chan error, 1),
	}
}

func (fw *fakeWatcher) Add(path string) error { return nil }
func (fw *fakeWatcher) Events() <-chan Event  { return fw.events }
func (fw *fakeWatcher) Errors() <-chan error  { return fw.errs }
func (fw *fakeWatcher) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if !fw.closed {
		fw.closed = true
		close(fw.events)
		close(fw.errs)
	}
	return nil
}

func (fw *fakeWatcher) emit(op Op, path string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if !fw.closed {
		fw.events <- Event{Path: path, Op: op}
	}
}

func TestSupervisorRestartsOnChange(t *testing.T) {
	fw := newFakeWatcher()
	defer func() { _ = fw.Close() }()

	restarts := make(chan struct{}, 3)
	restart := func(ctx context.Context) (int, error) {
		restarts <- struct{}{}
		<-ctx.Done()
		return 0, ctx.Err()
	}

	sup := NewSupervisor(SupervisorOptions{
		Watcher:          fw,
		WatchPaths:       []string{"/fake"},
		Restart:          restart,
		DebounceInterval: 10 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, err := sup.Run(ctx)
		errCh <- err
	}()

	// Wait for first restart to begin.
	select {
	case <-restarts:
		// First launch started.
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for first launch")
	}

	// Emit a file change.
	fw.emit(OpWrite, "/fake/app.ts")

	// Wait for the debounce and context cancellation.
	// The supervisor should cancel the first child, then restart.
	select {
	case <-restarts:
		// Second launch triggered by file change.
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for restart after file change")
	}

	cancel()
	<-errCh
}

func TestSupervisorDrainsOnCancel(t *testing.T) {
	fw := newFakeWatcher()
	defer func() { _ = fw.Close() }()

	started := make(chan struct{})
	restart := func(ctx context.Context) (int, error) {
		started <- struct{}{}
		<-ctx.Done()
		return 130, ctx.Err()
	}

	sup := NewSupervisor(SupervisorOptions{
		Watcher:          fw,
		WatchPaths:       []string{"/fake"},
		Restart:          restart,
		DebounceInterval: 10 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		_, err := sup.Run(ctx)
		errCh <- err
	}()

	// Wait for start.
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for start")
	}

	// Cancel should kill the child and clean up.
	cancel()
	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for supervisor exit")
	}
}

func TestSupervisorDebounce(t *testing.T) {
	fw := newFakeWatcher()
	defer func() { _ = fw.Close() }()

	var mu sync.Mutex
	restartCount := 0
	restart := func(ctx context.Context) (int, error) {
		mu.Lock()
		restartCount++
		mu.Unlock()
		<-ctx.Done()
		return 0, ctx.Err()
	}

	sup := NewSupervisor(SupervisorOptions{
		Watcher:          fw,
		WatchPaths:       []string{"/fake"},
		Restart:          restart,
		DebounceInterval: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_, err := sup.Run(ctx)
		_ = err
	}()

	// Wait for first start to settle.
	time.Sleep(50 * time.Millisecond)

	// Emit 5 rapid changes.
	for i := 0; i < 5; i++ {
		fw.emit(OpWrite, "/fake/x.ts")
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce window to close + restart.
	time.Sleep(300 * time.Millisecond)

	cancel()

	mu.Lock()
	n := restartCount
	mu.Unlock()

	// Should have restarted only once or twice, not 6 times.
	if n > 3 {
		t.Errorf("expected <= 3 restarts with debounce, got %d", n)
	}
}

func TestSupervisorClearScreen(t *testing.T) {
	// Verify the option is accepted without panic.
	sup := NewSupervisor(SupervisorOptions{
		ClearScreen: true,
	})
	if sup == nil {
		t.Fatal("nil supervisor")
	}
}
