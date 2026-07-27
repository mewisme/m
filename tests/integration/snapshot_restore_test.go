package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSnapshotRestoreAtomicSuccess(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "snap-atomic",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	lockBefore, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if code, out := runM(t, projDir, cfgPath, "add", "pkg-c"); code != 0 {
		t.Fatalf("add exit=%d out=%s", code, out)
	}
	if !hasDirectDep(t, projDir, "pkg-c") {
		t.Fatal("expected pkg-c after add")
	}
	if code, out := runM(t, projDir, cfgPath, "snapshot", "restore", "000001"); code != 0 {
		t.Fatalf("restore exit=%d out=%s", code, out)
	}
	if hasDirectDep(t, projDir, "pkg-c") {
		t.Fatal("pkg-c should be removed after restore")
	}
	lockAfter, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("lock should match pre-add snapshot")
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "pkg-a", "package.json")); err != nil {
		t.Fatal("pkg-a should be linked after atomic restore")
	}
}

func TestSnapshotRestoreMissingSnapshot(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "snap-missing",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	pkgBefore, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	lockBefore, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	code, out := runM(t, projDir, cfgPath, "snapshot", "restore", "999999")
	if code == 0 {
		t.Fatalf("expected failure, out=%s", out)
	}
	pkgAfter, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	lockAfter, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if string(pkgAfter) != string(pkgBefore) {
		t.Fatal("package.json changed after failed restore")
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("m.lock changed after failed restore")
	}
}

func TestSnapshotRestoreCorruptSnapshotIntegrity(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "snap-corrupt",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	if code, _ := runM(t, projDir, cfgPath, "add", "pkg-c"); code != 0 {
		t.Fatal("add failed")
	}
	snapLock := filepath.Join(projDir, ".mew", "snapshots", "000001", "m.lock")
	if err := os.WriteFile(snapLock, []byte("not-a-lock"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgBefore, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	lockBefore, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	code, out := runM(t, projDir, cfgPath, "snapshot", "restore", "000001")
	if code == 0 {
		t.Fatalf("expected integrity failure, out=%s", out)
	}
	pkgAfter, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	lockAfter, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if string(pkgAfter) != string(pkgBefore) {
		t.Fatal("package.json changed after failed restore")
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("m.lock changed after failed restore")
	}
	if !hasDirectDep(t, projDir, "pkg-c") {
		t.Fatal("live tree should remain at post-add state")
	}
}

func TestSnapshotRestoreFetchFailurePreservesLive(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "snap-fetch-fail",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	if code, _ := runM(t, projDir, cfgPath, "add", "pkg-c"); code != 0 {
		t.Fatal("add failed")
	}
	snapLock := filepath.Join(projDir, ".mew", "snapshots", "000001", "m.lock")
	lockData, err := os.ReadFile(snapLock)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(lockData), `"integrity":`, `"integrity":"sha512-deadbeef", "junk":`, 1)
	if tampered == string(lockData) {
		t.Fatal("could not tamper snapshot lock integrity")
	}
	if err := os.WriteFile(snapLock, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgBefore, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	lockBefore, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	code, _ := runM(t, projDir, cfgPath, "snapshot", "restore", "000001")
	if code == 0 {
		t.Fatal("expected fetch/verify failure during restore")
	}
	pkgAfter, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	lockAfter, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if string(pkgAfter) != string(pkgBefore) {
		t.Fatal("package.json changed after failed restore")
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("m.lock changed after failed restore")
	}
}

func TestSnapshotRestoreDigestMismatchPreservesLive(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "snap-digest",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	if code, _ := runM(t, projDir, cfgPath, "add", "pkg-c"); code != 0 {
		t.Fatal("add failed")
	}
	metaPath := filepath.Join(projDir, ".mew", "snapshots", "000001", "meta.json")
	meta, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(meta), `"graphDigest":`, `"graphDigest":"sha256:deadbeef", "junk":`, 1)
	if tampered == string(meta) {
		t.Fatal("could not tamper snapshot graphDigest")
	}
	if err := os.WriteFile(metaPath, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgBefore, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	lockBefore, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	code, out := runM(t, projDir, cfgPath, "snapshot", "restore", "000001")
	if code == 0 {
		t.Fatalf("expected digest mismatch failure, out=%s", out)
	}
	pkgAfter, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	lockAfter, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if string(pkgAfter) != string(pkgBefore) {
		t.Fatal("package.json changed after failed restore")
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("m.lock changed after failed restore")
	}
}

func TestSnapshotRestoreManifestLockDriftPreservesLive(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "snap-pair-drift",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, _ := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatal("install failed")
	}
	if code, _ := runM(t, projDir, cfgPath, "add", "pkg-c"); code != 0 {
		t.Fatal("add failed")
	}
	snapManifest := filepath.Join(projDir, ".mew", "snapshots", "000001", "package.json")
	manifestData, err := os.ReadFile(snapManifest)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(manifestData), `"dependencies"`, `"dependencies":{"pkg-z":"^1.0.0"}, "junkDependencies"`, 1)
	if tampered == string(manifestData) {
		t.Fatal("could not tamper snapshot manifest")
	}
	if err := os.WriteFile(snapManifest, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgBefore, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	lockBefore, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	code, out := runM(t, projDir, cfgPath, "snapshot", "restore", "000001")
	if code == 0 {
		t.Fatalf("expected manifest/lock drift failure, out=%s", out)
	}
	pkgAfter, _ := os.ReadFile(filepath.Join(projDir, "package.json"))
	lockAfter, _ := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if string(pkgAfter) != string(pkgBefore) {
		t.Fatal("package.json changed after failed restore")
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("m.lock changed after failed restore")
	}
}

func TestSnapshotRestoreDespiteLiveManifestLockDrift(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "snap-live-drift",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	lockBefore, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if code, out := runM(t, projDir, cfgPath, "add", "pkg-c"); code != 0 {
		t.Fatalf("add exit=%d out=%s", code, out)
	}
	livePkg := filepath.Join(projDir, "package.json")
	pkgData, err := os.ReadFile(livePkg)
	if err != nil {
		t.Fatal(err)
	}
	var pkgDoc map[string]json.RawMessage
	if err := json.Unmarshal(pkgData, &pkgDoc); err != nil {
		t.Fatal(err)
	}
	var deps map[string]string
	if err := json.Unmarshal(pkgDoc["dependencies"], &deps); err != nil {
		t.Fatal(err)
	}
	deps["pkg-d"] = "^1.0.0"
	depsBytes, err := json.Marshal(deps)
	if err != nil {
		t.Fatal(err)
	}
	pkgDoc["dependencies"] = depsBytes
	tampered, err := json.MarshalIndent(pkgDoc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	tampered = append(tampered, '\n')
	if err := os.WriteFile(livePkg, tampered, 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "snapshot", "restore", "000001")
	if code != 0 {
		t.Fatalf("restore should succeed despite live drift, exit=%d out=%s", code, out)
	}
	if hasDirectDep(t, projDir, "pkg-c") {
		t.Fatal("pkg-c should be removed after restore")
	}
	lockAfter, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("lock should match pre-add snapshot")
	}
}
