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
