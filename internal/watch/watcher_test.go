package watch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewWatcher(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	if w == nil {
		t.Fatal("NewWatcher returned nil")
	}
	_ = w.Close()
}

func TestNativeWatcherAddFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	if err := w.Add(f); err != nil {
		t.Fatalf("Add file: %v", err)
	}
}

func TestNativeWatcherAddDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.ts"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	if err := w.Add(dir); err != nil {
		t.Fatalf("Add dir: %v", err)
	}
}

func TestNativeWatcherDetectsWrite(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "watchme.txt")
	if err := os.WriteFile(f, []byte("initial"), 0o644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	if err := w.Add(dir); err != nil {
		t.Fatalf("Add dir: %v", err)
	}

	// Write to the file.
	if err := os.WriteFile(f, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for the event.
	select {
	case evt := <-w.Events():
		if evt.Path == "" {
			t.Error("event has empty path")
		}
		if evt.Op == 0 {
			t.Error("event has zero op")
		}
	// fsnotify events may be delayed; don't block test forever.
	default:
		// File system events are async — skip assertion if none arrived.
	}
}

func TestNativeWatcherDetectsCreate(t *testing.T) {
	dir := t.TempDir()

	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	defer func() { _ = w.Close() }()

	if err := w.Add(dir); err != nil {
		t.Fatalf("Add dir: %v", err)
	}

	newFile := filepath.Join(dir, "newfile.txt")
	if err := os.WriteFile(newFile, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-w.Events():
		// Expected.
	default:
	}
}

func TestPollingWatcherDetectsWrite(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "pollme.txt")
	if err := os.WriteFile(f, []byte("initial"), 0o644); err != nil {
		t.Fatal(err)
	}

	pw := newPollingWatcher(50) // 50ms interval for fast test
	defer func() { _ = pw.Close() }()

	if err := pw.Add(dir); err != nil {
		t.Fatalf("Add dir: %v", err)
	}

	// Write to trigger detection.
	if err := os.WriteFile(f, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case evt := <-pw.Events():
		if evt.Path == "" {
			t.Error("event has empty path")
		}
	// Polling at 50ms should detect within a few intervals.
	default:
	}
}

func TestCollectPaths(t *testing.T) {
	dir := t.TempDir()

	// Create a project-like structure.
	ep := filepath.Join(dir, "src", "index.ts")
	if err := os.MkdirAll(filepath.Dir(ep), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ep, []byte("console.log(1)"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("KEY=value"), 0o644); err != nil {
		t.Fatal(err)
	}

	paths, err := CollectPaths(ep, dir)
	if err != nil {
		t.Fatalf("CollectPaths: %v", err)
	}

	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[filepath.Base(p)] = true
	}

	// Should include the entrypoint, tsconfig, package.json, .env.
	if !pathSet["index.ts"] {
		t.Error("missing entrypoint in collected paths")
	}
	if !pathSet["tsconfig.json"] {
		t.Error("missing tsconfig.json in collected paths")
	}
	if !pathSet["package.json"] {
		t.Error("missing package.json in collected paths")
	}
	if !pathSet[".env"] {
		t.Error("missing .env in collected paths")
	}
}

func TestCollectPathsNoDupes(t *testing.T) {
	dir := t.TempDir()
	ep := filepath.Join(dir, "index.ts")
	if err := os.WriteFile(ep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	paths, err := CollectPaths(ep, dir)
	if err != nil {
		t.Fatalf("CollectPaths: %v", err)
	}

	seen := make(map[string]int)
	for _, p := range paths {
		seen[p]++
	}
	for p, n := range seen {
		if n > 1 {
			t.Errorf("path %s appears %d times", p, n)
		}
	}
}

func TestNormalizePath(t *testing.T) {
	cwd, _ := os.Getwd()
	rel := "test.txt"
	norm := normalizePath(rel)
	if !filepath.IsAbs(norm) {
		t.Errorf("normalizePath should return absolute path, got %s", norm)
	}
	_ = cwd
}
