package watch

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const defaultPollInterval = 500 * time.Millisecond

// pollingWatcher periodically stats watched files and emits events on modification.
type pollingWatcher struct {
	mu       sync.Mutex
	paths    map[string]time.Time // path -> last mod time
	interval time.Duration
	events   chan Event
	errs     chan error
	done     chan struct{}
	closed   bool
}

func newPollingWatcher(interval time.Duration) *pollingWatcher {
	if interval <= 0 {
		interval = defaultPollInterval
	}
	pw := &pollingWatcher{
		paths:    make(map[string]time.Time),
		interval: interval,
		events:   make(chan Event, 256),
		errs:     make(chan error, 16),
		done:     make(chan struct{}),
	}
	go pw.loop()
	return pw
}

func (pw *pollingWatcher) Add(path string) error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return pw.addDirLocked(path)
	}

	pw.paths[normalizePath(path)] = info.ModTime()
	return nil
}

func (pw *pollingWatcher) addDirLocked(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			switch base {
			case "node_modules", ".git", ".mew":
				return filepath.SkipDir
			}
			if isHiddenDir(base) {
				return filepath.SkipDir
			}
			return nil
		}
		pw.paths[normalizePath(path)] = info.ModTime()
		return nil
	})
}

func (pw *pollingWatcher) Remove(path string) error {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	norm := normalizePath(path)
	for k := range pw.paths {
		if k == norm || strings.HasPrefix(k, norm+string(filepath.Separator)) {
			delete(pw.paths, k)
		}
	}
	return nil
}

func (pw *pollingWatcher) Events() <-chan Event { return pw.events }
func (pw *pollingWatcher) Errors() <-chan error { return pw.errs }

func (pw *pollingWatcher) Close() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	if !pw.closed {
		pw.closed = true
		close(pw.done)
	}
	return nil
}

func (pw *pollingWatcher) loop() {
	ticker := time.NewTicker(pw.interval)
	defer ticker.Stop()
	defer close(pw.events)
	defer close(pw.errs)

	for {
		select {
		case <-pw.done:
			return
		case <-ticker.C:
			pw.poll()
		}
	}
}

func (pw *pollingWatcher) poll() {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	for path, lastMod := range pw.paths {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				delete(pw.paths, path)
				pw.events <- Event{Path: path, Op: OpRemove}
			}
			continue
		}
		if info.ModTime().After(lastMod) {
			pw.paths[path] = info.ModTime()
			pw.events <- Event{Path: path, Op: OpWrite}
		}
	}
}
