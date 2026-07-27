package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/testkit"
)

func TestValidateManifestPathHostile(t *testing.T) {
	cases := []struct {
		path string
	}{
		{""},
		{"../escape"},
		{"foo/../../etc"},
		{"/absolute"},
		{"foo/./bar"},
	}
	for _, tc := range cases {
		if err := validateManifestPath(tc.path); err == nil {
			t.Fatalf("path %q should be rejected", tc.path)
		}
	}
}

func TestValidateTreeManifestDuplicatePaths(t *testing.T) {
	m := &TreeManifest{
		SchemaVersion: 2,
		Entries: []TreeEntry{
			{Path: "a.txt", Kind: string(EntryFile), Hash: strings.Repeat("a", 64), Mode: 0o444},
			{Path: "a.txt", Kind: string(EntryFile), Hash: strings.Repeat("b", 64), Mode: 0o444},
		},
	}
	if err := validateTreeManifest(m); err == nil {
		t.Fatal("expected duplicate path error")
	}
}

func TestValidateTreeManifestCaseCollision(t *testing.T) {
	m := &TreeManifest{
		SchemaVersion: 2,
		Entries: []TreeEntry{
			{Path: "lib/Util.js", Kind: string(EntryFile), Hash: strings.Repeat("a", 64), Mode: 0o444},
			{Path: "lib/util.js", Kind: string(EntryFile), Hash: strings.Repeat("b", 64), Mode: 0o444},
		},
	}
	if err := validateTreeManifest(m); err == nil {
		t.Fatal("expected case collision error")
	} else if !strings.Contains(err.Error(), "portable path collision") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateTreeManifestTrailingDotCollision(t *testing.T) {
	m := &TreeManifest{
		SchemaVersion: 2,
		Entries: []TreeEntry{
			{Path: "readme.md", Kind: string(EntryFile), Hash: strings.Repeat("a", 64), Mode: 0o444},
			{Path: "readme.md.", Kind: string(EntryFile), Hash: strings.Repeat("b", 64), Mode: 0o444},
		},
	}
	if err := validateTreeManifest(m); err == nil {
		t.Fatal("expected trailing dot collision error")
	} else if !strings.Contains(err.Error(), "portable path collision") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateTreeManifestTrailingSpaceCollision(t *testing.T) {
	m := &TreeManifest{
		SchemaVersion: 2,
		Entries: []TreeEntry{
			{Path: "data.json", Kind: string(EntryFile), Hash: strings.Repeat("a", 64), Mode: 0o444},
			{Path: "data.json ", Kind: string(EntryFile), Hash: strings.Repeat("b", 64), Mode: 0o444},
		},
	}
	if err := validateTreeManifest(m); err == nil {
		t.Fatal("expected trailing space collision error")
	}
}

func TestPortableCollisionKeySlashNormalization(t *testing.T) {
	a := portableCollisionKey("lib/foo")
	b := portableCollisionKey(`lib\foo`)
	if a != b {
		t.Fatalf("keys differ: %q vs %q", a, b)
	}
}

func TestVerifyTreeManifestDiskCaseCollision(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "probe"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "PROBE"), []byte("b"), 0o644); err != nil {
		t.Skip("case-insensitive filesystem")
	}
	probe, err := os.ReadFile(filepath.Join(dir, "probe"))
	if err != nil || string(probe) != "a" {
		t.Skip("case-insensitive filesystem")
	}
	_ = os.Remove(filepath.Join(dir, "probe"))
	_ = os.Remove(filepath.Join(dir, "PROBE"))

	pkgData := []byte(`{"name":"x"}`)
	pkgPath := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkgPath, pkgData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, packageMarker), []byte("sha256-dead"), 0o444); err != nil {
		t.Fatal(err)
	}
	altPath := filepath.Join(dir, "PACKAGE.json")
	if err := os.WriteFile(altPath, []byte("collision"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	m := &TreeManifest{
		SchemaVersion: 2,
		Entries: []TreeEntry{
			{Path: "package.json", Kind: string(EntryFile), Hash: mustHashBytes(pkgData), Mode: uint32(info.Mode().Perm())},
		},
	}
	if err := writeTreeManifest(dir, m); err != nil {
		t.Fatal(err)
	}
	if err := verifyTreeManifest(dir, m); err == nil {
		t.Fatal("expected disk case collision failure")
	} else if !strings.Contains(err.Error(), "portable path collision") {
		t.Fatalf("err=%v", err)
	}
}

func TestVerifyTreeManifestExtraFile(t *testing.T) {
	dir := t.TempDir()
	pkgData := []byte(`{"name":"x"}`)
	pkgPath := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkgPath, pkgData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, packageMarker), []byte("sha256-dead"), 0o444); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	m := &TreeManifest{
		SchemaVersion: 2,
		Entries: []TreeEntry{
			{Path: "package.json", Kind: string(EntryFile), Hash: mustHashBytes(pkgData), Mode: uint32(info.Mode().Perm())},
		},
	}
	if err := writeTreeManifest(dir, m); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyTreeManifest(dir, m); err == nil {
		t.Fatal("expected extra file failure")
	} else if !strings.Contains(err.Error(), "extra file") {
		t.Fatalf("err=%v", err)
	}
}

func TestVerifyTreeManifestTypeSwap(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	pkgData := []byte(`{"name":"x"}`)
	pkgPath := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkgPath, pkgData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, packageMarker), []byte("sha256-dead"), 0o444); err != nil {
		t.Fatal(err)
	}
	libInfo, err := os.Lstat(filepath.Join(dir, "lib"))
	if err != nil {
		t.Fatal(err)
	}
	pkgInfo, err := os.Lstat(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	m := &TreeManifest{
		SchemaVersion: 2,
		Entries: []TreeEntry{
			{Path: "lib", Kind: string(EntryDir), Mode: uint32(libInfo.Mode().Perm())},
			{Path: "package.json", Kind: string(EntryFile), Hash: mustHashBytes(pkgData), Mode: uint32(pkgInfo.Mode().Perm())},
		},
	}
	if err := writeTreeManifest(dir, m); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dir, "lib")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lib"), []byte("not-a-dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyTreeManifest(dir, m); err == nil {
		t.Fatal("expected type swap failure")
	}
}

func TestVerifyTreeManifestSymlinkTargetChange(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("symlink test skipped on windows")
	}
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(targetPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink("target.txt", link); err != nil {
		t.Fatal(err)
	}
	pkgData := []byte(`{"name":"x"}`)
	pkgPath := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkgPath, pkgData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, packageMarker), []byte("sha256-dead"), 0o444); err != nil {
		t.Fatal(err)
	}
	linkInfo, _ := os.Lstat(link)
	targetInfo, _ := os.Lstat(targetPath)
	pkgInfo, _ := os.Lstat(pkgPath)
	m := &TreeManifest{
		SchemaVersion: 2,
		Entries: []TreeEntry{
			{Path: "link", Kind: string(EntrySymlink), Mode: uint32(linkInfo.Mode().Perm()), SymlinkTarget: "target.txt"},
			{Path: "package.json", Kind: string(EntryFile), Hash: mustHashBytes(pkgData), Mode: uint32(pkgInfo.Mode().Perm())},
			{Path: "target.txt", Kind: string(EntryFile), Hash: mustHashBytes([]byte("x")), Mode: uint32(targetInfo.Mode().Perm())},
		},
	}
	if err := writeTreeManifest(dir, m); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(link)
	if err := os.Symlink("other.txt", link); err != nil {
		t.Fatal(err)
	}
	if err := verifyTreeManifest(dir, m); err == nil {
		t.Fatal("expected symlink target failure")
	}
}

func TestLegacyPackageRequiresReimport(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	ps := NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "pkg-cli-1.0.0.tgz")
	integrity := "sha256-6ffb2697417ee0f02ad400c8d92c46cfb5889cf84603cd1f797146fde316b5d0"
	key, err := PackageKeyFromIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}
	pkgDir := ps.PackagePath(key)
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"legacy"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writePackageMarker(pkgDir, key); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ps.VerifyPackage(ctx, key); err == nil {
		t.Fatal("expected legacy verify failure")
	} else if !strings.Contains(err.Error(), "re-import tarball") {
		t.Fatalf("err=%v", err)
	}
	if _, err := importIntegrity(ctx, ps, tgz, integrity); err != nil {
		t.Fatal(err)
	}
	if err := ps.VerifyPackage(ctx, key); err != nil {
		t.Fatal(err)
	}
}

func TestHostileManifestPathInFile(t *testing.T) {
	dir := t.TempDir()
	raw := `{"schemaVersion":2,"entries":[{"path":"../escape","kind":"file","hash":"` + strings.Repeat("a", 64) + `","mode":420}]}`
	if err := os.WriteFile(treeManifestPath(dir), []byte(raw), 0o444); err != nil {
		t.Fatal(err)
	}
	m, err := readTreeManifest(dir)
	if err == nil {
		t.Fatal("expected hostile path rejection on read")
	}
	if !strings.Contains(err.Error(), "escaping manifest path") && !strings.Contains(err.Error(), "store.manifest") {
		t.Fatalf("err=%v", err)
	}
	_ = m
}

func TestTreeManifestIncludesDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lib", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := generateTreeManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, e := range m.Entries {
		if e.Kind == string(EntryDir) {
			found[e.Path] = true
		}
	}
	if !found["lib"] || !found["lib/nested"] {
		t.Fatalf("dirs missing from manifest: %+v", m.Entries)
	}
}

func mustHashBytes(b []byte) string {
	f := filepath.Join(os.TempDir(), "mew-hash-test")
	if err := os.WriteFile(f, b, 0o644); err != nil {
		panic(err)
	}
	defer func() { _ = os.Remove(f) }()
	h, err := hashFile(f)
	if err != nil {
		panic(err)
	}
	return h
}

func TestVerifyRejectsMalformedManifestJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(treeManifestPath(dir), []byte("{"), 0o444); err != nil {
		t.Fatal(err)
	}
	_, err := readTreeManifest(dir)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestUnsupportedManifestEntryKind(t *testing.T) {
	m := &TreeManifest{
		SchemaVersion: 2,
		Entries: []TreeEntry{
			{Path: "weird", Kind: "socket", Mode: 0o644},
		},
	}
	if err := validateTreeManifest(m); err == nil {
		t.Fatal("expected unsupported kind error")
	}
}

func TestRejectSchemaVersionZero(t *testing.T) {
	dir := t.TempDir()
	raw := `{"schemaVersion":0,"entries":[]}`
	if err := os.WriteFile(treeManifestPath(dir), []byte(raw), 0o444); err != nil {
		t.Fatal(err)
	}
	_, err := readTreeManifest(dir)
	if err == nil {
		t.Fatal("expected schema version rejection")
	}
}

func TestValidateManifestHashRejectsShort(t *testing.T) {
	m := &TreeManifest{
		SchemaVersion: 2,
		Entries: []TreeEntry{
			{Path: "a.txt", Kind: string(EntryFile), Hash: "dead", Mode: 0o444},
		},
	}
	if err := validateTreeManifest(m); err == nil {
		t.Fatal("expected short hash rejection")
	}
}

func TestValidateManifestPathWindowsReserved(t *testing.T) {
	for _, path := range []string{"CON", "lib/PRN", "pkg/COM1"} {
		if err := validateManifestPath(path); err == nil {
			t.Fatalf("path %q should be rejected", path)
		}
	}
}

func TestValidateSymlinkTargetEscapes(t *testing.T) {
	for _, target := range []string{"../outside", "/abs", ".."} {
		if err := validateSymlinkTarget(target); err == nil {
			t.Fatalf("target %q should be rejected", target)
		}
	}
}

func TestValidateSymlinkTargetWindowsReserved(t *testing.T) {
	if err := validateSymlinkTarget("CON"); err == nil {
		t.Fatal("expected reserved target rejection")
	}
}

func TestRejectUnsupportedManifestSchema(t *testing.T) {
	m := &TreeManifest{
		SchemaVersion: 1,
		Entries:       []TreeEntry{},
	}
	if err := validateTreeManifest(m); err == nil {
		t.Fatal("expected unsupported schema rejection")
	}
}

func TestTreeManifestJSONRoundTrip(t *testing.T) {
	m := &TreeManifest{
		SchemaVersion: 2,
		Entries: []TreeEntry{
			{Path: "a", Kind: string(EntryFile), Hash: strings.Repeat("c", 64), Mode: 0o444},
		},
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var got TreeManifest
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
}
