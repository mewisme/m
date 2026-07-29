package archive_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/archive"
	"github.com/mewisme/mew/internal/testkit"
)

func TestPatchRejectsTraversal(t *testing.T) {
	root := newPatchPackageRoot(t)
	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(outside) }()

	beforeRoot := hashDir(t, root)
	beforeOutside := fileHash(t, outside)

	patch := writePatch(t, []byte(`diff --git a/../outside.txt b/../outside.txt
index 0000000..0000000 100644
--- a/../outside.txt
+++ b/../outside.txt
@@ -1 +1 @@
-outside
+owned
`))
	err := archive.ApplyUnifiedPatch(context.Background(), root, patch)
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("expected integrity error, got %v", err)
	}
	assertTreeUnchanged(t, root, beforeRoot)
	assertFileUnchanged(t, outside, beforeOutside)
}

func TestPatchRejectsAbsolutePath(t *testing.T) {
	root := newPatchPackageRoot(t)
	outside := filepath.Join(t.TempDir(), "absolute-target.txt")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	beforeRoot := hashDir(t, root)
	beforeOutside := fileHash(t, outside)

	patch := writePatch(t, []byte(`diff --git a/`+filepath.ToSlash(outside)+` b/`+filepath.ToSlash(outside)+`
--- a/`+filepath.ToSlash(outside)+`
+++ b/`+filepath.ToSlash(outside)+`
@@ -1 +1 @@
-x
+y
`))
	err := archive.ApplyUnifiedPatch(context.Background(), root, patch)
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("expected integrity error, got %v", err)
	}
	assertTreeUnchanged(t, root, beforeRoot)
	assertFileUnchanged(t, outside, beforeOutside)
}

func TestPatchRejectsCreateDelete(t *testing.T) {
	root := newPatchPackageRoot(t)
	beforeRoot := hashDir(t, root)

	createPatch := writePatch(t, []byte(`diff --git a/new.js b/new.js
new file mode 100644
--- /dev/null
+++ b/new.js
@@ -0,0 +1 @@
+created
`))
	if err := archive.ApplyUnifiedPatch(context.Background(), root, createPatch); apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("create: got %v", err)
	}
	assertTreeUnchanged(t, root, beforeRoot)

	deletePatch := writePatch(t, []byte(`diff --git a/index.js b/index.js
deleted file mode 100644
--- a/index.js
+++ /dev/null
@@ -1 +0,0 @@
-original
`))
	if err := archive.ApplyUnifiedPatch(context.Background(), root, deletePatch); apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("delete: got %v", err)
	}
	assertTreeUnchanged(t, root, beforeRoot)
}

func TestPatchRejectsRename(t *testing.T) {
	root := newPatchPackageRoot(t)
	beforeRoot := hashDir(t, root)
	patch := writePatch(t, []byte(`diff --git a/index.js b/renamed.js
index 0000000..0000000 100644
--- a/index.js
+++ b/renamed.js
@@ -1 +1 @@
-original
+renamed
`))
	if err := archive.ApplyUnifiedPatch(context.Background(), root, patch); apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("got %v", err)
	}
	assertTreeUnchanged(t, root, beforeRoot)
}

func TestPatchRejectsDuplicateTargets(t *testing.T) {
	root := newPatchPackageRoot(t)
	beforeRoot := hashDir(t, root)
	patch := writePatch(t, []byte(`diff --git a/index.js b/index.js
index 0000000..0000000 100644
--- a/index.js
+++ b/index.js
@@ -1 +1 @@
-original
+one
diff --git a/index.js b/index.js
index 0000000..0000000 100644
--- a/index.js
+++ b/index.js
@@ -1 +1 @@
-original
+two
`))
	if err := archive.ApplyUnifiedPatch(context.Background(), root, patch); apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("got %v", err)
	}
	assertTreeUnchanged(t, root, beforeRoot)
}

func TestPatchAllowsQuotedPathWithSpaces(t *testing.T) {
	root := newPatchPackageRoot(t)
	spaced := filepath.Join(root, "path with spaces", "file.js")
	if err := os.MkdirAll(filepath.Dir(spaced), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spaced, []byte("line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := writePatch(t, []byte(`diff --git a/path with spaces/file.js b/path with spaces/file.js
index 0000000..0000000 100644
--- a/path with spaces/file.js
+++ b/path with spaces/file.js
@@ -1 +1 @@
-line
+patched
`))
	if err := archive.ApplyUnifiedPatch(context.Background(), root, patch); err != nil {
		t.Fatalf("apply: %v", err)
	}
	data, err := os.ReadFile(spaced)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "patched\n" {
		t.Fatalf("got %q", data)
	}
}

func TestPatchRejectsSymlinkTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevated privileges on Windows")
	}
	root := newPatchPackageRoot(t)
	outside := filepath.Join(filepath.Dir(root), "symlink-outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(outside) }()
	link := filepath.Join(root, "link.js")
	if err := os.Symlink(outside, link); err != nil {
		t.Skip(err)
	}
	beforeRoot := hashDir(t, root)
	beforeOutside := fileHash(t, outside)

	patch := writePatch(t, []byte(`diff --git a/link.js b/link.js
index 0000000..0000000 100644
--- a/link.js
+++ b/link.js
@@ -1 +1 @@
-outside
+owned
`))
	if err := archive.ApplyUnifiedPatch(context.Background(), root, patch); apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("got %v", err)
	}
	assertTreeUnchanged(t, root, beforeRoot)
	assertFileUnchanged(t, outside, beforeOutside)
}

func TestPatchRejectsDrivePathOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("drive-qualified paths are windows-specific")
	}
	root := newPatchPackageRoot(t)
	beforeRoot := hashDir(t, root)
	patch := writePatch(t, []byte(`diff --git a/C:/Windows/win.ini b/C:/Windows/win.ini
--- a/C:/Windows/win.ini
+++ b/C:/Windows/win.ini
@@ -1 +1 @@
-x
+y
`))
	if err := archive.ApplyUnifiedPatch(context.Background(), root, patch); apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("got %v", err)
	}
	assertTreeUnchanged(t, root, beforeRoot)
}

func TestPatchRejectsUNCPathOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("unc paths are windows-specific")
	}
	root := newPatchPackageRoot(t)
	beforeRoot := hashDir(t, root)
	_, err := archive.ResolvePatchTargetForTest("patch.patch", root, "\\\\server\\share\\file.txt")
	if err == nil {
		t.Fatal("expected unc rejection")
	}
	assertTreeUnchanged(t, root, beforeRoot)
}

func TestPatchPathErrorFields(t *testing.T) {
	root := newPatchPackageRoot(t)
	patch := writePatch(t, []byte(`diff --git a/../escape.js b/../escape.js
--- a/../escape.js
+++ b/../escape.js
@@ -1 +1 @@
-x
+y
`))
	err := archive.ApplyUnifiedPatch(context.Background(), root, patch)
	var ppe *archive.PatchPathError
	if !errors.As(err, &ppe) {
		t.Fatalf("expected PatchPathError, got %T %v", err, err)
	}
	if ppe.PatchFile == "" || ppe.PackageRoot == "" || ppe.Original == "" {
		t.Fatalf("missing fields: %+v", ppe)
	}
}

func TestMsFixturePatchApply(t *testing.T) {
	fixture := testkit.FixtureDir(t, "locks/sources/pnpm/patch")
	extract := filepath.Join(t.TempDir(), "extract")
	if err := os.MkdirAll(extract, 0o755); err != nil {
		t.Fatal(err)
	}
	msIndex := readMsIndexForPatch(t)
	if err := os.WriteFile(filepath.Join(extract, "index.js"), []byte(msIndex), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(fixture, "patches", "ms@2.1.3.patch")
	if err := archive.ApplyUnifiedPatch(context.Background(), extract, patch); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(extract, "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(data), "ms-patched") {
		t.Fatalf("patch not applied: %q", data)
	}
}

func readMsIndexForPatch(t *testing.T) string {
	t.Helper()
	path := filepath.Join(testkit.ModuleRoot(t), "internal", "archive", "testdata", "ms-2.1.3-index.js")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

func newPatchPackageRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "pkg")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.js"), []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func writePatch(t *testing.T, body []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "patch.patch")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func hashDir(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if _, err := h.Write([]byte(rel + "\n")); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		if _, err := h.Write(sum[:]); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func fileHash(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func assertTreeUnchanged(t *testing.T, root, before string) {
	t.Helper()
	if got := hashDir(t, root); got != before {
		t.Fatalf("package tree changed")
	}
}

func assertFileUnchanged(t *testing.T, path, before string) {
	t.Helper()
	if got := fileHash(t, path); got != before {
		t.Fatalf("probe file %s changed", path)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
