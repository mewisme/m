package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/snapshot"
)

func TestSnapshotWorkspaceCapturesMemberManifests(t *testing.T) {
	projDir, cfgPath := setupWorkspaceProject(t, "projects/workspace-filter")
	code, out := runM(t, projDir, cfgPath, "install", "-r")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	list, err := snapshot.NewStore(projDir).List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("expected snapshot after install")
	}
	latest := list[0]
	want := map[string]bool{
		"packages/alpha/package.json": false,
		"packages/beta/package.json":  false,
	}
	for _, rel := range latest.MemberManifests {
		rel = filepath.ToSlash(rel)
		if _, ok := want[rel]; ok {
			want[rel] = true
		}
	}
	for rel, ok := range want {
		if !ok {
			t.Fatalf("snapshot missing member manifest %s: %v", rel, latest.MemberManifests)
		}
	}
	rec, err := snapshot.NewStore(projDir).Load(latest.ID)
	if err != nil {
		t.Fatal(err)
	}
	betaLive, err := os.ReadFile(filepath.Join(projDir, "packages", "beta", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(rec.MemberManifests[filepath.ToSlash(filepath.Join("packages", "beta", "package.json"))]) != string(betaLive) {
		t.Fatal("snapshot member bytes diverge from live beta package.json")
	}
}

func TestSnapshotWorkspaceMemberRestore(t *testing.T) {
	projDir, cfgPath := setupWorkspaceProject(t, "projects/workspace-filter")
	code, out := runM(t, projDir, cfgPath, "install", "-r")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	betaPath := filepath.Join(projDir, "packages", "beta", "package.json")
	betaBefore, err := os.ReadFile(betaPath)
	if err != nil {
		t.Fatal(err)
	}
	list, err := snapshot.NewStore(projDir).List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("expected snapshot after install")
	}
	if err := os.WriteFile(betaPath, []byte(`{"name":"beta","version":"9.9.9"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out = runM(t, projDir, cfgPath, "snapshot", "restore", list[0].ID)
	if code != 0 {
		t.Fatalf("restore exit=%d out=%s", code, out)
	}
	betaRestored, err := os.ReadFile(betaPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(betaRestored) != string(betaBefore) {
		t.Fatalf("beta not restored from snapshot:\nwant %s\ngot %s", betaBefore, betaRestored)
	}
}
