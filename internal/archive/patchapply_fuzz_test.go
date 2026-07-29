package archive_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/archive"
)

func FuzzPatchPreflight(f *testing.F) {
	f.Add("diff --git a/index.js b/index.js\n--- a/index.js\n+++ b/index.js\n@@ -1 +1 @@\n-line\n+patched\n")
	f.Add("diff --git a/../outside b/../outside\n--- a/../outside\n+++ b/../outside\n@@ -1 +1 @@\n-a\n+b\n")
	f.Add("diff --git a/C:/evil b/C:/evil\n--- a/C:/evil\n+++ b/C:/evil\n@@ -1 +1 @@\n-a\n+b\n")
	f.Fuzz(func(t *testing.T, patchBody string) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "index.js"), []byte("line\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		patchPath := filepath.Join(t.TempDir(), "fuzz.patch")
		if err := os.WriteFile(patchPath, []byte(patchBody), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := archive.PreflightPlan(context.Background(), patchPath, root)
		if err != nil {
			if apperr.CodeOf(err) == apperr.Cancelled {
				return
			}
			if strings.Contains(patchBody, "../") && apperr.CodeOf(err) != apperr.Integrity {
				t.Fatalf("traversal patch should fail integrity: %v", err)
			}
			return
		}
		if err := archive.ApplyUnifiedPatch(context.Background(), root, patchPath); err != nil {
			if apperr.CodeOf(err) == apperr.Cancelled || apperr.CodeOf(err) == apperr.Integrity {
				return
			}
			t.Fatalf("unexpected apply error: %v", err)
		}
	})
}

func FuzzPatchCancelledContext(f *testing.F) {
	f.Add("diff --git a/index.js b/index.js\n--- a/index.js\n+++ b/index.js\n@@ -1 +1 @@\n-line\n+patched\n")
	f.Fuzz(func(t *testing.T, patchBody string) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "index.js"), []byte("line\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		patchPath := filepath.Join(t.TempDir(), "fuzz.patch")
		if err := os.WriteFile(patchPath, []byte(patchBody), 0o644); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := archive.ApplyUnifiedPatch(ctx, root, patchPath); err == nil {
			t.Fatal("expected cancellation")
		}
	})
}
