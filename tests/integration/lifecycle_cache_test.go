package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/lifecycle"
)

func TestLifecyclePrepareRerunsAfterNodeModulesRemoval(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node required for prepare test")
	}
	projDir, cfgPath := setupLifecycleProject(t, "lifecycle-counter")
	if code, out := runM(t, projDir, cfgPath, "trust", "lifecycle-counter"); code != 0 {
		t.Fatalf("trust exit=%d out=%s", code, out)
	}
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("first install exit=%d out=%s", code, out)
	}
	auditPath := lifecycle.AuditFilePath(projDir)
	first, err := lifecycle.ReadAudit(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if countPrepare(first, "lifecycle-counter") != 1 {
		t.Fatalf("want one prepare audit entry, got %+v", first)
	}
	if err := os.Remove(filepath.Join(projDir, "m.lock")); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(projDir, "node_modules")); err != nil {
		t.Fatal(err)
	}
	code, out = runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("second install exit=%d out=%s", code, out)
	}
	second, err := lifecycle.ReadAudit(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if countPrepare(second, "lifecycle-counter") != 2 {
		t.Fatalf("prepare must re-run when outputs are gone; audit=%+v", second)
	}
}

func countPrepare(entries []lifecycle.AuditEntry, pkg string) int {
	n := 0
	for _, e := range entries {
		if e.Package == pkg && e.Script == "prepare" {
			n++
		}
	}
	return n
}
