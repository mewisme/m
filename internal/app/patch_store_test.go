package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/archive"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/testkit"
)

func TestStagePatchDerivativesCopiesStoreTree(t *testing.T) {
	ctx := context.Background()
	storeRoot := filepath.Join(t.TempDir(), "store")
	src := filepath.Join(storeRoot, "pkg", "ms@2.1.3")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(msPackageIndex)
	if err := os.WriteFile(filepath.Join(src, "index.js"), original, 0o644); err != nil {
		t.Fatal(err)
	}
	before := dirDigest(t, src)

	pkgKey := "ms@2.1.3(patch_hash=abc)"
	patchPath := writePatchFixture(t)
	ext := patchExtensions(t, pkgKey, patchPath, "abc")
	extracts := map[string]string{pkgKey: src}
	stage := t.TempDir()

	if err := stagePatchDerivatives(ctx, stage, storeRoot, ext, extracts); err != nil {
		t.Fatal(err)
	}
	if extracts[pkgKey] == src {
		t.Fatal("expected derivative path")
	}
	if after := dirDigest(t, src); after != before {
		t.Fatal("store tree mutated during staging")
	}
	if err := applyPatchesToExtracts(ctx, graphWithPatch(pkgKey), ext, extracts); err != nil {
		t.Fatal(err)
	}
	if after := dirDigest(t, src); after != before {
		t.Fatal("store tree mutated during patch apply")
	}
	data, err := os.ReadFile(filepath.Join(extracts[pkgKey], "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "ms-patched") {
		t.Fatalf("derivative not patched: %q", data)
	}
}

func TestStagePatchDerivativesDistinctWritableRoots(t *testing.T) {
	ctx := context.Background()
	storeRoot := filepath.Join(t.TempDir(), "store")
	src := filepath.Join(storeRoot, "pkg", "ms@2.1.3")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "index.js"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stage := t.TempDir()
	keyA := "ms@2.1.3(patch_hash=aaaaaaaaaaaaaaaa)"
	keyB := "ms@2.1.3(patch_hash=bbbbbbbbbbbbbbbb)"
	ext := lockfile.Extensions{}
	raw, err := json.Marshal(map[string]resolver.PatchSource{
		keyA: {Path: writePatchFixture(t), Hash: "aaaaaaaaaaaaaaaa"},
		keyB: {Path: writePatchFixture(t), Hash: "bbbbbbbbbbbbbbbb"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ext[resolver.PatchExtensionKey] = raw
	extracts := map[string]string{keyA: src, keyB: src}
	if err := stagePatchDerivatives(ctx, stage, storeRoot, ext, extracts); err != nil {
		t.Fatal(err)
	}
	if extracts[keyA] == extracts[keyB] {
		t.Fatal("expected distinct derivative directories")
	}
}

func TestValidatePatchPlanMissingExtract(t *testing.T) {
	patches := map[string]resolver.PatchSource{
		"ms@2.1.3(patch_hash=x)": {Path: writePatchFixture(t), Hash: "x"},
	}
	err := validatePatchPlan(graphWithPatch("ms@2.1.3(patch_hash=x)"), patches, map[string]string{})
	if err == nil {
		t.Fatal("expected missing extract error")
	}
}

func TestValidatePatchPlanMissingPatchFile(t *testing.T) {
	pkgKey := "ms@2.1.3(patch_hash=x)"
	patches := map[string]resolver.PatchSource{
		pkgKey: {Path: filepath.Join(t.TempDir(), "missing.patch"), Hash: "x"},
	}
	err := validatePatchPlan(graphWithPatch(pkgKey), patches, map[string]string{pkgKey: t.TempDir()})
	if err == nil {
		t.Fatal("expected missing patch file error")
	}
}

func TestApplyPatchesRollbackLeavesStoreUntouched(t *testing.T) {
	ctx := context.Background()
	storeRoot := filepath.Join(t.TempDir(), "store")
	src := filepath.Join(storeRoot, "pkg", "ms@2.1.3")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "index.js"), []byte(msPackageIndex), 0o644); err != nil {
		t.Fatal(err)
	}
	before := dirDigest(t, src)
	pkgKey := "ms@2.1.3(patch_hash=abc)"
	badPatch := filepath.Join(t.TempDir(), "bad.patch")
	if err := os.WriteFile(badPatch, []byte(`diff --git a/missing.js b/missing.js
--- a/missing.js
+++ b/missing.js
@@ -1 +1 @@
-x
+y
`), 0o644); err != nil {
		t.Fatal(err)
	}
	ext := patchExtensions(t, pkgKey, badPatch, "abc")
	extracts := map[string]string{pkgKey: src}
	if err := stagePatchDerivatives(ctx, t.TempDir(), storeRoot, ext, extracts); err != nil {
		t.Fatal(err)
	}
	if err := applyPatchesToExtracts(ctx, graphWithPatch(pkgKey), ext, extracts); err == nil {
		t.Fatal("expected patch apply failure")
	}
	if after := dirDigest(t, src); after != before {
		t.Fatal("store mutated after failed patch apply")
	}
}

func writePatchFixture(t *testing.T) string {
	t.Helper()
	src := testkit.FixtureDir(t, "locks/sources/pnpm/patch/patches/ms@2.1.3.patch")
	dest := filepath.Join(t.TempDir(), "ms.patch")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dest
}

func patchExtensions(t *testing.T, pkgKey, patchPath, hash string) lockfile.Extensions {
	t.Helper()
	raw, err := json.Marshal(map[string]resolver.PatchSource{
		pkgKey: {Path: patchPath, Hash: hash},
	})
	if err != nil {
		t.Fatal(err)
	}
	return lockfile.Extensions{resolver.PatchExtensionKey: raw}
}

func graphWithPatch(pkgKey string) *graph.Graph {
	i := strings.Index(pkgKey, "@")
	if i < 0 {
		return &graph.Graph{}
	}
	return &graph.Graph{
		Packages: []graph.Package{{ID: graph.PackageID{
			Name:    pkgKey[:i],
			Version: pkgKey[i+1:],
		}}},
	}
}

const msPackageIndex = `/**
 * Helpers.
 */

var s = 1000;
var m = s * 60;
var h = m * 60;
var d = h * 24;
var w = d * 7;
var y = d * 365.25;

/**
 * Parse or format the given ` + "`val`" + `.
 *
 * Options:
 *
 *  - ` + "`long`" + ` verbose formatting [false]
 *
 * @param {String|Number} val
 * @param {Object} [options]
 * @throws {Error} throw an error if val is not a non-empty string or a number
 * @return {String|Number}
 * @api public
 */

module.exports = function (val, options) {
  options = options || {};
  var type = typeof val;
  if (type === 'string' && val.length > 0) {
    return parse(val);
  } else if (type === 'number' && isFinite(val)) {
    return options.long ? fmtLong(val) : fmtShort(val);
  }
  throw new Error(
    'val is not a non-empty string or a valid number. val=' +
      JSON.stringify(val)
  );
};
`

func dirDigest(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		h.Write([]byte(rel))
		h.Write(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestLegacyExtractSkipsStoreCopy(t *testing.T) {
	ctx := context.Background()
	extract := filepath.Join(t.TempDir(), "extract", "ms")
	if err := os.MkdirAll(extract, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extract, "index.js"), []byte(msPackageIndex), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgKey := "ms@2.1.3(patch_hash=abc)"
	ext := patchExtensions(t, pkgKey, writePatchFixture(t), "abc")
	extracts := map[string]string{pkgKey: extract}
	if err := stagePatchDerivatives(ctx, t.TempDir(), "/global/store", ext, extracts); err != nil {
		t.Fatal(err)
	}
	if extracts[pkgKey] != extract {
		t.Fatalf("legacy extract path changed to %q", extracts[pkgKey])
	}
	if err := applyPatchesToExtracts(ctx, graphWithPatch(pkgKey), ext, extracts); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(extract, "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "ms-patched") {
		t.Fatalf("expected patched content, got %q", data)
	}
}

func TestArchiveCopyDirTreeUsedByStage(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := archive.CopyDirTree(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "a" {
		t.Fatalf("copy failed: %q", data)
	}
}
