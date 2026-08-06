// Package watch provides file watching and a restart supervisor.
package watch

import (
	"os"
	"path/filepath"
	"strings"
)

// Op describes a file operation.
type Op uint32

const (
	OpCreate Op = 1 << iota
	OpWrite
	OpRemove
	OpRename
)

// Event is a file change notification.
type Event struct {
	Path string
	Op   Op
}

// Watcher watches files and directories for changes.
type Watcher interface {
	// Add starts watching a path (file or directory, recursively).
	Add(path string) error
	// Events returns the event channel.
	Events() <-chan Event
	// Errors returns the error channel (non-fatal errors).
	Errors() <-chan error
	// Close stops watching and releases resources.
	Close() error
}

// NewWatcher creates the best available watcher for the current platform.
func NewWatcher() (Watcher, error) {
	return newNativeWatcher()
}

// CollectPaths gathers filesystem paths to watch for a project:
// entrypoint, tsconfig.json, package.json, .env files, and
// the entrypoint's directory for source changes.
func CollectPaths(entrypoint, cwd string) ([]string, error) {
	cwd, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}

	var paths []string
	seen := make(map[string]bool)

	addPath := func(p string) {
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return
		}
		if real, err := filepath.EvalSymlinks(abs); err == nil {
			abs = real
		}
		if seen[abs] {
			return
		}
		seen[abs] = true
		paths = append(paths, abs)
	}

	if entrypoint != "" {
		epAbs := entrypoint
		if !filepath.IsAbs(epAbs) {
			epAbs = filepath.Join(cwd, epAbs)
		}
		addPath(epAbs)
		addPath(filepath.Dir(epAbs))
	}

	for dir := cwd; dir != "" && dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		tsc := filepath.Join(dir, "tsconfig.json")
		if _, err := os.Stat(tsc); err == nil {
			addPath(tsc)
		}
		pkg := filepath.Join(dir, "package.json")
		if _, err := os.Stat(pkg); err == nil {
			addPath(pkg)
		}
		envPatterns := []string{".env", ".env.local", ".env.development", ".env.production"}
		for _, name := range envPatterns {
			envPath := filepath.Join(dir, name)
			if _, err := os.Stat(envPath); err == nil {
				addPath(envPath)
			}
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			n := e.Name()
			if strings.HasPrefix(n, ".env.") && strings.HasSuffix(n, ".local") {
				addPath(filepath.Join(dir, n))
			}
		}
	}

	return paths, nil
}

func isHiddenDir(name string) bool {
	return strings.HasPrefix(name, ".") && name != "." && name != ".."
}

func normalizePath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	return abs
}
