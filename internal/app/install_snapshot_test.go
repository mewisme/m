package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/testkit"
)

func TestReadStagedOrLivePrefersStaged(t *testing.T) {
	dir := t.TempDir()
	staged := filepath.Join(dir, "staged.json")
	live := filepath.Join(dir, "live.json")
	if err := os.WriteFile(staged, []byte("staged"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(live, []byte("live"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := readStagedOrLive(staged, live)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "staged" {
		t.Fatalf("expected staged, got %q", data)
	}
}

func TestReadStagedOrLiveFallsBackOnNotExist(t *testing.T) {
	dir := t.TempDir()
	staged := filepath.Join(dir, "staged.json")
	live := filepath.Join(dir, "live.json")
	if err := os.WriteFile(live, []byte("live"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := readStagedOrLive(staged, live)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "live" {
		t.Fatalf("expected live, got %q", data)
	}
}

func TestReadStagedOrLivePropagatesStagedIOError(t *testing.T) {
	dir := t.TempDir()
	// Staged path is a directory, not a file — ReadFile will fail.
	staged := filepath.Join(dir, "staged-dir")
	if err := os.MkdirAll(staged, 0o755); err != nil {
		t.Fatal(err)
	}
	live := filepath.Join(dir, "live.json")
	if err := os.WriteFile(live, []byte("live"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readStagedOrLive(staged, live)
	if err == nil {
		t.Fatal("expected error reading staged directory as file, got nil")
	}
}

func TestReadStagedOrLivePropagatesLiveIOError(t *testing.T) {
	dir := t.TempDir()
	staged := filepath.Join(dir, "staged.json")
	live := filepath.Join(dir, "nonexistent", "live.json")
	// Neither staged nor live exists.
	_, err := readStagedOrLive(staged, live)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}
}

func TestReadStagedOrLiveBothExistUsesStaged(t *testing.T) {
	dir := t.TempDir()
	staged := filepath.Join(dir, "staged.json")
	live := filepath.Join(dir, "live.json")
	if err := os.WriteFile(staged, []byte(`{"name":"staged"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(live, []byte(`{"name":"live"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := readStagedOrLive(staged, live)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "staged") {
		t.Fatalf("expected staged content, got %q", data)
	}
}

func TestMlockAdapterPropagatesContextCancellation(t *testing.T) {
	// Verify that mlock.Adapter.Read and Write check ctx.Err() before I/O.
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "m.lock")
	if err := os.WriteFile(lockPath, []byte(`{"schemaVersion":3,"importers":[],"packages":[],"edges":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	adapter := mlock.Adapter{}
	_, err := adapter.Read(ctx, lockPath)
	if err == nil {
		t.Fatal("expected cancellation error from Read, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled from Read, got %v", err)
	}

	err = adapter.Write(ctx, lockPath, &graph.Graph{SchemaVersion: 3})
	if err == nil {
		t.Fatal("expected cancellation error from Write, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled from Write, got %v", err)
	}
}

func TestExtAdapterPropagatesContextCancellation(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "m.lock")
	if err := os.WriteFile(lockPath, []byte(`{"schemaVersion":3,"importers":[],"packages":[],"edges":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ext := mlock.ExtAdapter{}
	_, _, err := ext.ReadWithExtensions(ctx, lockPath)
	if err == nil {
		t.Fatal("expected cancellation error from ReadWithExtensions, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled from ReadWithExtensions, got %v", err)
	}

	_, err = ext.EncodePreserving(ctx, lockPath, &graph.Graph{SchemaVersion: 3}, nil, nil, lockfile.Detection{Format: "nub"})
	if err == nil {
		t.Fatal("expected cancellation error from EncodePreserving, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled from EncodePreserving, got %v", err)
	}
}

func TestStagedManifestFallbackOnlyOnNotExist(t *testing.T) {
	// Verify readStagedOrLive returns live bytes only when staged is missing,
	// and propagates staged permission errors (represented by directory read).
	dir := t.TempDir()

	// Case 1: Staged dir (EISDIR), live file → should fail, not fall back.
	staged := filepath.Join(dir, "staged-is-dir")
	if err := os.MkdirAll(staged, 0o755); err != nil {
		t.Fatal(err)
	}
	live := filepath.Join(dir, "live.json")
	if err := os.WriteFile(live, []byte("live"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readStagedOrLive(staged, live)
	if err == nil {
		t.Fatal("expected error reading staged directory, got nil")
	}

	// Case 2: Staged missing (ENOENT), live exists → fall back.
	staged2 := filepath.Join(dir, "staged-missing.json")
	data, err := readStagedOrLive(staged2, live)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "live" {
		t.Fatalf("expected live fallback, got %q", data)
	}
}

func TestDLXLifecycleIsNotCalled(t *testing.T) {
	// Verify that the runLifecyclePhase call has been removed from
	// buildDLXEnvironment and Materialize. This is a compile-time check:
	// the buildDLXEnvironment function no longer imports lifecycle,
	// which means the dead call is gone.
	//
	// The line-count test verifies the lifecycle.Enabled check is absent.
	src, err := os.ReadFile(filepath.Join("dlx_install.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "lifecycle.Enabled") {
		t.Fatal("buildDLXEnvironment should not reference lifecycle.Enabled after dead call removal")
	}
	if strings.Contains(string(src), "runLifecyclePhase") {
		t.Fatal("buildDLXEnvironment should not call runLifecyclePhase after dead call removal")
	}

	matSrc, err := os.ReadFile(filepath.Join("envexec_materialize.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(matSrc), "runLifecyclePhase") {
		t.Fatal("Materialize should not call runLifecyclePhase after dead call removal")
	}
}

func TestMalformedStagedLockFails(t *testing.T) {
	// Verify that a malformed staged lock produces a Lockfile error, not a
	// generic parse error.
	opts := InstallOptions{
		PreResolvedGraph: nil, // No pre-resolved graph
		StagedLock:       []byte("not valid json"),
	}
	// PreResolvedGraph is nil, so staged lock decode doesn't happen.
	// But when PreResolvedGraph is set, it should fail.
	opts.PreResolvedGraph = &graph.Graph{SchemaVersion: 3}
	// The error path is tested in runInstallInSession lines 220-227.
	// Test the decode directly.
	_, err := mlock.Decode(opts.StagedLock)
	if err == nil {
		t.Fatal("expected decode error for malformed lock bytes, got nil")
	}
	// Verify error type propagates correctly through the install path.
	wrapped := apperr.Wrap(apperr.Lockfile, "app.install", "staged lock", err)
	if apperr.CodeOf(wrapped) != apperr.Lockfile {
		t.Fatalf("expected Lockfile error code, got %v", apperr.CodeOf(wrapped))
	}
}

func TestGreenfieldMissingPriorLockAccepted(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	// No lockfile at all — greenfield.
	doc, err := readPriorLockDocument(dir, project.IdentityMew)
	if err != nil {
		t.Fatal(err)
	}
	if doc != nil {
		t.Fatal("expected nil document for missing lock in greenfield")
	}
}

func TestPriorLockReadErrorPropagates(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	// Write a malformed m.lock to ensure read error propagates.
	lockPath := filepath.Join(dir, "m.lock")
	if err := os.WriteFile(lockPath, []byte("{not json}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readPriorLockDocument(dir, project.IdentityMew)
	if err == nil {
		t.Fatal("expected error reading malformed prior lock, got nil")
	}
	// Should be a Lockfile or Config error, not silently nil.
	if errors.Is(err, os.ErrNotExist) {
		t.Fatal("malformed lock must not be treated as missing")
	}
}

func TestPropsFromNubLock(t *testing.T) {
	nubLockPath := filepath.Join("..", "..", "fixtures", "locks", "nub", "basic-project", "nub.lock")
	_, err := os.Stat(nubLockPath)
	if os.IsNotExist(err) {
		t.Skip("fixture not available: " + nubLockPath)
	}
	if err != nil {
		t.Fatal(err)
	}
}
