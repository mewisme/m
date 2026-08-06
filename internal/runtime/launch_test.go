package runtime_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/runtime"
)

func TestMergeCleanupError_BothNil(t *testing.T) {
	err := runtime.MergeCleanupError(nil, nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestMergeCleanupError_LaunchSuccessCleanupFails(t *testing.T) {
	cleanupErr := errors.New("cleanup boom")
	err := runtime.MergeCleanupError(nil, cleanupErr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("expected cleanup error, got %v", err)
	}
}

func TestMergeCleanupError_LaunchFailsCleanupSuccess(t *testing.T) {
	launchErr := errors.New("launch boom")
	err := runtime.MergeCleanupError(launchErr, nil)
	if !errors.Is(err, launchErr) {
		t.Fatalf("expected launch error, got %v", err)
	}
}

func TestMergeCleanupError_BothFail(t *testing.T) {
	launchErr := errors.New("launch boom")
	cleanupErr := errors.New("cleanup boom")
	err := runtime.MergeCleanupError(launchErr, cleanupErr)
	if err == nil {
		t.Fatal("expected error")
	}
	// Primary (launch) must be preserved.
	if !errors.Is(err, launchErr) {
		t.Fatalf("primary should be launch error, got %v", err)
	}
	// Cleanup must also be reachable.
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("cleanup error not found in chain, got %v", err)
	}
}

func TestMergeCleanupError_PreservesChildExitCode(t *testing.T) {
	exitStatus := &apperr.ExitStatus{Code: 42, Err: errors.New("exit 42")}
	cleanupErr := errors.New("cleanup boom")
	err := runtime.MergeCleanupError(exitStatus, cleanupErr)
	// CodeOf must resolve to ChildExit (through Primary).
	if apperr.CodeOf(err) != apperr.ChildExit {
		t.Fatalf("expected ChildExit code, got %s", apperr.CodeOf(err))
	}
	// ExitCode must return 42.
	if apperr.ExitCode(err) != 42 {
		t.Fatalf("expected exit code 42, got %d", apperr.ExitCode(err))
	}
}

func TestMergeCleanupError_PreservesCancellation(t *testing.T) {
	cancelErr := apperr.Wrap(apperr.Cancelled, "runtime.launch", "test.js", context.Canceled)
	cleanupErr := errors.New("cleanup boom")
	err := runtime.MergeCleanupError(cancelErr, cleanupErr)
	if apperr.CodeOf(err) != apperr.Cancelled {
		t.Fatalf("expected Cancelled code, got %s", apperr.CodeOf(err))
	}
	// Both errors must be in the chain.
	if !errors.Is(err, cancelErr) {
		t.Fatalf("cancellation not preserved as primary")
	}
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("cleanup error not in chain")
	}
}

func TestMergeCleanupError_ErrorFormat(t *testing.T) {
	launchErr := fmt.Errorf("launch: %w", errors.New("inner"))
	cleanupErr := fmt.Errorf("cleanup: %w", errors.New("inner2"))
	err := runtime.MergeCleanupError(launchErr, cleanupErr)
	msg := err.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
	// Must contain launch error message.
	if !contains(msg, "launch") {
		t.Fatalf("error message missing launch detail: %q", msg)
	}
	// Must contain cleanup error.
	if !contains(msg, "cleanup") {
		t.Fatalf("error message missing cleanup detail: %q", msg)
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
