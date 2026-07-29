package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func TestSourcesFileDepInstallAndCI(t *testing.T) {
	projDir := t.TempDir()
	testkit.CopyFixture(t, "sources/file-dep", projDir)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testkit.CleanEnv(t)

	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	pkgJSON := filepath.Join(projDir, "node_modules", "vendor-pkg", "package.json")
	if _, err := os.Stat(pkgJSON); err != nil {
		t.Fatalf("vendor-pkg not linked: %v", err)
	}
	lockBefore, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if code, out := runM(t, projDir, cfgPath, "ci"); code != 0 {
		t.Fatalf("ci exit=%d out=%s", code, out)
	}
	lockAfter, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if string(lockAfter) != string(lockBefore) {
		t.Fatal("m.lock changed after ci")
	}
}

func TestSourcesGitDepPinnedCommit(t *testing.T) {
	if _, err := os.Stat(filepath.Join(testkit.ModuleRoot(t), "fixtures", "sources", "git-dep")); err != nil {
		t.Skip("git-dep fixture missing")
	}
	projDir := t.TempDir()
	testkit.CopyFixture(t, "sources/git-dep", projDir)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testkit.CleanEnv(t)

	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	pkgJSON := filepath.Join(projDir, "node_modules", "git-sample-pkg", "package.json")
	if _, err := os.Stat(pkgJSON); err != nil {
		t.Fatalf("git package not linked: %v", err)
	}
	lockBytes, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(lockBytes), "mew.resolver/git") || !contains(string(lockBytes), "1e92b302cc5df841ccc7a74c7d88e8d2c2e13535") {
		t.Fatalf("lock missing git extension metadata: %s", string(lockBytes))
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
