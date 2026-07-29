package archive_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/archive"
	"github.com/mewisme/mew/internal/transaction"
)

func TestApplyPatchAtomicPublishesOnSuccess(t *testing.T) {
	source := newAtomicSource(t)
	work := filepath.Join(t.TempDir(), "work")
	publish := filepath.Join(t.TempDir(), "publish")
	patch := writeAtomicPatch(t)

	opts := archive.ApplyPatchOptions{
		SourceRoot:  source,
		WorkRoot:    work,
		PublishRoot: publish,
		PatchPath:   patch,
	}
	if err := archive.ApplyPatchAtomic(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(work); err == nil {
		t.Fatal("work dir should be renamed away")
	}
	data, err := os.ReadFile(filepath.Join(publish, "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "patched\n" {
		t.Fatalf("got %q", data)
	}
	srcData, err := os.ReadFile(filepath.Join(source, "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(srcData) != "original\n" {
		t.Fatalf("source mutated: %q", srcData)
	}
}

func TestApplyPatchAtomicFailureLeavesSourceAndPublish(t *testing.T) {
	source := newAtomicSource(t)
	work := filepath.Join(t.TempDir(), "work")
	publish := filepath.Join(t.TempDir(), "publish")
	if err := os.MkdirAll(publish, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(publish, "index.js"), []byte("publish\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := writeTraversalPatch(t)

	opts := archive.ApplyPatchOptions{
		SourceRoot:  source,
		WorkRoot:    work,
		PublishRoot: publish,
		PatchPath:   patch,
	}
	if err := archive.ApplyPatchAtomic(context.Background(), opts); err == nil {
		t.Fatal("expected failure")
	}
	if _, err := os.Stat(work); !os.IsNotExist(err) {
		t.Fatalf("work dir should be removed, stat err=%v", err)
	}
	data, err := os.ReadFile(filepath.Join(publish, "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "publish\n" {
		t.Fatalf("publish changed: %q", data)
	}
}

func TestApplyPatchAtomicHookInjection(t *testing.T) {
	phases := []string{"post_patch_copy", "post_patch_preflight", "post_patch_apply", "post_patch_publish"}
	for _, phase := range phases {
		t.Run(phase, func(t *testing.T) {
			source := newAtomicSource(t)
			work := filepath.Join(t.TempDir(), "work")
			publish := filepath.Join(t.TempDir(), "publish")
			patch := writeAtomicPatch(t)

			transaction.SetTestHook(func(hookPhase string, _ int) error {
				if hookPhase == phase {
					return os.ErrPermission
				}
				return nil
			})
			t.Cleanup(func() { transaction.SetTestHook(nil) })

			opts := archive.ApplyPatchOptions{
				SourceRoot:  source,
				WorkRoot:    work,
				PublishRoot: publish,
				PatchPath:   patch,
			}
			if err := archive.ApplyPatchAtomic(context.Background(), opts); err == nil {
				t.Fatal("expected hook failure")
			}
			if _, err := os.Stat(publish); !os.IsNotExist(err) {
				t.Fatalf("publish should not exist after %s failure", phase)
			}
			src, err := os.ReadFile(filepath.Join(source, "index.js"))
			if err != nil {
				t.Fatal(err)
			}
			if string(src) != "original\n" {
				t.Fatalf("source mutated after %s", phase)
			}
		})
	}
}

func newAtomicSource(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.js"), []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeAtomicPatch(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "patch.patch")
	body := `diff --git a/index.js b/index.js
index 0000000..0000000 100644
--- a/index.js
+++ b/index.js
@@ -1 +1 @@
-original
+patched
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeTraversalPatch(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bad.patch")
	body := `diff --git a/../outside.txt b/../outside.txt
--- a/../outside.txt
+++ b/../outside.txt
@@ -1 +1 @@
-x
+y
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
