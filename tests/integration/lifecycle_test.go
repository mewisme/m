package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/lifecycle"
)

func skipWithoutNode(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node required")
	}
}

func TestLifecyclePostinstallWritesMarker(t *testing.T) {
	skipWithoutNode(t)
	projDir, cfgPath := setupLifecycleProject(t, "lifecycle-postinstall-ok")
	if code, out := runM(t, projDir, cfgPath, "trust", "lifecycle-postinstall-ok"); code != 0 {
		t.Fatalf("trust exit=%d out=%s", code, out)
	}
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	marker := filepath.Join(projDir, "node_modules", "lifecycle-postinstall-ok", "marker.txt")
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "ok" {
		t.Fatalf("marker missing: err=%v data=%q", err, data)
	}
}

func TestLifecycleIgnoreScriptsSkipsExecution(t *testing.T) {
	projDir, cfgPath := setupLifecycleProject(t, "lifecycle-postinstall-ok")
	if code, out := runM(t, projDir, cfgPath, "trust", "lifecycle-postinstall-ok"); code != 0 {
		t.Fatalf("trust exit=%d out=%s", code, out)
	}
	code, out := runM(t, projDir, cfgPath, "install", "--ignore-scripts")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	marker := filepath.Join(projDir, "node_modules", "lifecycle-postinstall-ok", "marker.txt")
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("marker should not exist with --ignore-scripts")
	}
}

func TestLifecycleUntrustedBlocksUntilTrust(t *testing.T) {
	projDir, cfgPath := setupLifecycleProject(t, "lifecycle-postinstall-ok")
	code, out := runM(t, projDir, cfgPath, "install")
	if code == 0 {
		t.Fatalf("expected untrusted failure, out=%s", out)
	}
	if code, out = runM(t, projDir, cfgPath, "trust", "lifecycle-postinstall-ok"); code != 0 {
		t.Fatalf("trust exit=%d out=%s", code, out)
	}
	code, out = runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("install after trust exit=%d out=%s", code, out)
	}
}

func TestLifecycleAuditRecordsExecution(t *testing.T) {
	skipWithoutNode(t)
	projDir, cfgPath := setupLifecycleProject(t, "lifecycle-postinstall-ok")
	if code, out := runM(t, projDir, cfgPath, "trust", "lifecycle-postinstall-ok"); code != 0 {
		t.Fatalf("trust exit=%d out=%s", code, out)
	}
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	entries, err := lifecycle.ReadAudit(lifecycle.AuditFilePath(projDir))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if e.Package == "lifecycle-postinstall-ok" && e.Script == "postinstall" && e.ExitCode == 0 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing audit entry, got %+v", entries)
	}
}

func TestBuildsListShowsAudit(t *testing.T) {
	skipWithoutNode(t)
	projDir, cfgPath := setupLifecycleProject(t, "lifecycle-postinstall-ok")
	if code, out := runM(t, projDir, cfgPath, "trust", "lifecycle-postinstall-ok"); code != 0 {
		t.Fatalf("trust exit=%d out=%s", code, out)
	}
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	code, out := runM(t, projDir, cfgPath, "builds", "list")
	if code != 0 {
		t.Fatalf("builds list exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "lifecycle-postinstall-ok") || !strings.Contains(out, "postinstall") {
		t.Fatalf("builds list missing audit: %s", out)
	}
}
