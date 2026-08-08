package watch

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const defaultPollInterval = 500 * time.Millisecond

// fileFingerprint is a cheap identity snapshot that does not depend solely
// on ModTime. A deleted path records a zero-value fingerprint so it can be
// rediscovered on recreate.
type fileFingerprint struct {
	size    int64
	modTime time.Time
	exists  bool
}

func fingerprint(info os.FileInfo) fileFingerprint {
	return fileFingerprint{size: info.Size(), modTime: info.ModTime(), exists: true}
}

// pollingWatcher periodically stats watched files and emits events on
// modification. It tracks logical roots so directories can be rescanned
// for new files.
type pollingWatcher struct {
	mu       sync.Mutex
	paths    map[string]fileFingerprint // canonical path -> last fingerprint
	roots    map[string]struct{}        // watched directories (logical roots)
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
		paths:    make(map[string]fileFingerprint),
		roots:    make(map[string]struct{}),
		interval: interval,
		events:   make(chan Event, 256),
		errs:     make(chan error, 16),
		done:     make(chan struct{}),
	}
	go pw.loop()
	return pw
}

func (pw *pollingWatcher) Backend() Backend { return BackendPolling }

func (pw *pollingWatcher) Add(path string) error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	key := pathKey(path)
	if info.IsDir() {
		pw.roots[key] = struct{}{}
		return pw.addDirLocked(path)
	}

	pw.paths[key] = fingerprint(info)
	return nil
}

func (pw *pollingWatcher) addDirLocked(dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if shouldSkipDir(base) {
				return filepath.SkipDir
			}
			pw.roots[pathKey(path)] = struct{}{}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		pw.paths[pathKey(path)] = fingerprint(info)
		return nil
	})
}

func (pw *pollingWatcher) Remove(path string) error {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	key := pathKey(path)
	delete(pw.roots, key)
	sep := string(filepath.Separator)
	for k := range pw.paths {
		if k == key || strings.HasPrefix(k, key+sep) {
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
	// Collect events under the lock, then emit outside it so a full
	// event channel cannot deadlock Close.
	var events []Event

	pw.mu.Lock()
	// Rescan each logical root for new files.
	for root := range pw.roots {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				base := filepath.Base(path)
				if shouldSkipDir(base) {
					return filepath.SkipDir
				}
				pw.roots[pathKey(path)] = struct{}{}
				return nil
			}
			key := pathKey(path)
			norm := normalizePath(path)
			info, err := d.Info()
			if err != nil {
				return nil
			}
			fp := fingerprint(info)
			prev, known := pw.paths[key]
			if !known {
				// New file discovered after initial snapshot.
				pw.paths[key] = fp
				events = append(events, Event{Path: norm, Op: OpCreate})
				return nil
			}
			if fp != prev {
				pw.paths[key] = fp
				events = append(events, Event{Path: norm, Op: OpWrite})
			}
			return nil
		})
	}

	// Check existing paths for deletion or modification (for files
	// outside logical roots).
	for path, prev := range pw.paths {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				// Keep the path with a zero-value fingerprint so we
				// detect recreate.
				pw.paths[path] = fileFingerprint{}
				events = append(events, Event{Path: path, Op: OpRemove})
			}
			continue
		}
		if info.IsDir() {
			continue // directories tracked via roots, not paths
		}
		fp := fingerprint(info)
		if fp != prev {
			pw.paths[path] = fp
			// If the previous fingerprint was a deletion tombstone,
			// emit create; otherwise emit write.
			if !prev.exists {
				events = append(events, Event{Path: path, Op: OpCreate})
			} else {
				events = append(events, Event{Path: path, Op: OpWrite})
			}
		}
	}
	pw.mu.Unlock()

	// Emit events outside the lock.
	for _, evt := range events {
		select {
		case pw.events <- evt:
		case <-pw.done:
			return
		}
	}
}
