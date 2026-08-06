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

// --- BuildArgv credential ordering ---

func TestBuildArgvCredentialPreloadFirst(t *testing.T) {
	credPreload := &runtime.PreloadAsset{Path: "/cache/credential-grabber.cjs", ModuleType: "cjs"}
	otherPreload := &runtime.PreloadAsset{Path: "/cache/preload.cjs", ModuleType: "cjs"}

	plan := &runtime.LaunchPlan{
		NodeExe:           "node",
		CredentialPreload: credPreload,
		PreloadAssets:     []runtime.PreloadAsset{*otherPreload},
		Entrypoint:        "app.js",
	}

	argv := runtime.BuildArgv(plan, []string{"--require", "/user/preload.js"})
	// Node processes --require left to right. Credential grabber must be FIRST.
	// Expected: node --require <cred-grabber> --require /user/preload.js --require <preload> app.js

	found := false
	for i, a := range argv {
		if a == "--require" && i+1 < len(argv) && argv[i+1] == credPreload.Path {
			// Check that it comes before user args.
			userIdx := -1
			for j, b := range argv {
				if b == "/user/preload.js" {
					userIdx = j
					break
				}
			}
			if userIdx >= 0 && i < userIdx {
				found = true
			}
			break
		}
	}
	if !found {
		t.Fatalf("credential grabber not first preload in argv: %v", argv)
	}
}

func TestBuildArgvZeroAugmentationNoCredentialPreload(t *testing.T) {
	plan := &runtime.LaunchPlan{
		NodeExe:          "node",
		PreloadAssets:    nil,
		Entrypoint:       "app.js",
		ZeroAugmentation: true,
	}
	plan.CredentialPreload = nil // explicit

	argv := runtime.BuildArgv(plan, nil)
	for _, a := range argv {
		if a == "--require" || a == "--import" {
			t.Fatalf("zero-augmentation mode injected preload: %v", argv)
		}
	}
}

func TestBuildArgvUserArgsAfterCredentialPreload(t *testing.T) {
	cred := &runtime.PreloadAsset{Path: "/c/cred.cjs", ModuleType: "cjs"}
	plan := &runtime.LaunchPlan{
		NodeExe:           "node",
		CredentialPreload: cred,
		PreloadAssets:     []runtime.PreloadAsset{{Path: "/c/loader.mjs", ModuleType: "esm"}},
		Entrypoint:        "app.ts",
	}
	v8Args := []string{"--require", "/user/evil.js", "--max-old-space-size=4096"}

	argv := runtime.BuildArgv(plan, v8Args)

	credIdx := -1
	userIdx := -1
	for i, a := range argv {
		if a == cred.Path {
			credIdx = i
		}
		if a == "/user/evil.js" {
			userIdx = i
		}
	}
	if credIdx < 0 || userIdx < 0 {
		t.Fatalf("could not find expected args in argv: %v", argv)
	}
	if credIdx >= userIdx {
		t.Fatalf("credential grabber (idx %d) must be before user preload (idx %d): %v", credIdx, userIdx, argv)
	}
}

// --- validateNodeEnv ---

func TestValidateNodeEnvRejectsRequire(t *testing.T) {
	err := runtime.ValidateNodeEnv([]string{"NODE_OPTIONS=--require ./evil.js"})
	if err == nil {
		t.Fatal("expected error for NODE_OPTIONS with --require")
	}
}

func TestValidateNodeEnvRejectsImport(t *testing.T) {
	err := runtime.ValidateNodeEnv([]string{"NODE_OPTIONS=--import ./evil.mjs"})
	if err == nil {
		t.Fatal("expected error for NODE_OPTIONS with --import")
	}
}

func TestValidateNodeEnvAllowsSafeOptions(t *testing.T) {
	err := runtime.ValidateNodeEnv([]string{"NODE_OPTIONS=--max-old-space-size=4096 --no-warnings"})
	if err != nil {
		t.Fatalf("unexpected error for safe NODE_OPTIONS: %v", err)
	}
}

func TestValidateNodeEnvNoNodeOptions(t *testing.T) {
	err := runtime.ValidateNodeEnv([]string{"PATH=/usr/bin", "HOME=/home/user"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateNodeEnvEmptyEnv(t *testing.T) {
	err := runtime.ValidateNodeEnv(nil)
	if err != nil {
		t.Fatalf("unexpected error for nil env: %v", err)
	}
}

// --- contains helper ---

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
