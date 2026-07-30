package fsx_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

func TestRenameErrorPreservesPaths(t *testing.T) {
	src := `F:\project\node_modules`
	dst := `F:\project\node_modules.mew-old`
	cause := errors.New("Access is denied.")
	re := fsx.NewRenameError("rename", src, dst, cause)
	wrapped := apperr.Wrap(apperr.IO, "transaction.publish", dst, re)
	msg := wrapped.Error()
	for _, want := range []string{"transaction.publish", src, dst, "Access is denied"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in %q", want, msg)
		}
	}
	if strings.HasSuffix(strings.TrimSpace(msg), "...") {
		t.Fatalf("truncated error: %q", msg)
	}
}

func TestRenamePathReplacesExistingDirectory(t *testing.T) {
	root := t.TempDir()
	live := filepath.Join(root, "node_modules")
	stage := filepath.Join(root, "stage_nm")
	if err := os.MkdirAll(filepath.Join(live, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(stage, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := fsx.PublishDirectory(stage, live); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(live, "pkg")); err != nil {
		t.Fatal(err)
	}
}

func TestRenamePathRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := fsx.RenamePath(ctx, filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
