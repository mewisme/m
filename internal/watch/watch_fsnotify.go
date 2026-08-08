package watch

import (
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type nativeWatcher struct {
	w      *fsnotify.Watcher
	events chan Event
	errs   chan error
	done   chan struct{}
}

func newNativeWatcher() (*nativeWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	nw := &nativeWatcher{
		w:      w,
		events: make(chan Event, 256),
		errs:   make(chan error, 16),
		done:   make(chan struct{}),
	}
	go nw.loop()
	return nw, nil
}

func (nw *nativeWatcher) Add(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		// fsnotify watches directories recursively by default.
		// Skip hidden directories during initial add.
		return nw.addDir(path)
	}
	return nw.w.Add(path)
}

func (nw *nativeWatcher) addDir(dir string) error {
	if err := nw.w.Add(dir); err != nil {
		return err
	}
	// Walk subdirectories, skipping hidden dirs and noise trees.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // non-fatal: dir may have been removed
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if isHiddenDir(name) {
			continue
		}
		if name == "node_modules" || name == ".git" || name == ".mew" {
			continue
		}
		sub := filepath.Join(dir, name)
		if err := nw.addDir(sub); err != nil {
			// Non-fatal: permission errors, etc.
			continue
		}
	}
	return nil
}

func (nw *nativeWatcher) Remove(path string) error { return nw.w.Remove(path) }

func (nw *nativeWatcher) Events() <-chan Event { return nw.events }
func (nw *nativeWatcher) Errors() <-chan error { return nw.errs }

func (nw *nativeWatcher) Close() error {
	close(nw.done)
	return nw.w.Close()
}

func (nw *nativeWatcher) loop() {
	defer close(nw.events)
	defer close(nw.errs)

	for {
		select {
		case <-nw.done:
			return
		case evt, ok := <-nw.w.Events:
			if !ok {
				return
			}
			op := toOp(evt.Op)
			// Skip root-only events for directories; fsnotify may emit
			// CHMOD on directories which are noise for our use case.
			if evt.Has(fsnotify.Chmod) && isDir(evt.Name) {
				continue
			}
			// Normalize the path for consistent event identity.
			path := normalizePath(evt.Name)

			// When a new directory is created under a watched dir,
			// add it to the watcher for recursive coverage.
			if evt.Has(fsnotify.Create) && isDir(evt.Name) && !isHiddenDir(filepath.Base(evt.Name)) {
				_ = nw.w.Add(path)
			}

			nw.events <- Event{Path: path, Op: op}
		case err, ok := <-nw.w.Errors:
			if !ok {
				return
			}
			nw.errs <- err
		}
	}
}

func toOp(op fsnotify.Op) Op {
	var o Op
	if op&fsnotify.Create != 0 {
		o |= OpCreate
	}
	if op&fsnotify.Write != 0 {
		o |= OpWrite
	}
	if op&fsnotify.Remove != 0 {
		o |= OpRemove
	}
	if op&fsnotify.Rename != 0 {
		o |= OpRename
	}
	return o
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// AddRecursive adds a directory and all subdirectories to the watcher,
// skipping hidden directories. Useful for project source trees.
func (nw *nativeWatcher) AddRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		if !info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if isHiddenDir(base) && path != dir {
			return filepath.SkipDir
		}
		// Skip node_modules and .git.
		if base == "node_modules" || base == ".git" || base == ".mew" {
			return filepath.SkipDir
		}
		if err := nw.w.Add(path); err != nil {
			return nil // skip unwatchable dirs
		}
		return nil
	})
}
