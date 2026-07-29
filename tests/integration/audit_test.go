package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/advisory"
	"github.com/mewisme/mew/internal/testkit"
)

func setupAuditProject(t *testing.T) (projDir, cfgPath string, env testkit.CleanEnvInfo) {
	t.Helper()
	env = testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := testkit.LoadRegistry(t, "registry/audit/v1")
	srv := reg.Start(t)

	projDir = t.TempDir()
	src := filepath.Join(testkit.ModuleRoot(t), "fixtures", "audit", "vulnerable-transitive", "package.json")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath = filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return projDir, cfgPath, env
}

func seedAdvisoryCache(t *testing.T, cacheDir string) {
	t.Helper()
	store := advisory.Store{Dir: filepath.Join(cacheDir, "advisory")}
	if err := store.SeedFixture(testkit.ModuleRoot(t)); err != nil {
		t.Fatal(err)
	}
}

func TestAuditReportsCVEOffline(t *testing.T) {
	projDir, cfgPath, env := setupAuditProject(t)
	seedAdvisoryCache(t, env.CacheDir)

	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}

	code, out := runM(t, projDir, cfgPath, "audit", "--offline")
	if code != 0 {
		t.Fatalf("audit exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "CVE-2026-0001") {
		t.Fatalf("missing CVE in output: %s", out)
	}
	if !strings.Contains(out, "vuln-pkg@1.0.0") {
		t.Fatalf("missing vuln-pkg in output: %s", out)
	}
}

func TestAuditJSONOffline(t *testing.T) {
	projDir, cfgPath, env := setupAuditProject(t)
	seedAdvisoryCache(t, env.CacheDir)

	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}

	code, out := runM(t, projDir, cfgPath, "audit", "--json", "--offline")
	if code != 0 {
		t.Fatalf("audit exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, `"CVE-2026-0001"`) {
		t.Fatalf("missing CVE in json: %s", out)
	}
	if !strings.Contains(out, `"schemaVersion"`) {
		t.Fatalf("missing schemaVersion: %s", out)
	}
}

func TestAuditFixOffline(t *testing.T) {
	projDir, cfgPath, env := setupAuditProject(t)
	seedAdvisoryCache(t, env.CacheDir)

	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}

	code, out := runM(t, projDir, cfgPath, "audit", "--fix", "--offline")
	if code != 0 {
		t.Fatalf("audit exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "vuln-pkg") || !strings.Contains(out, "1.0.1") {
		t.Fatalf("missing fix suggestion: %s", out)
	}
}

func TestAuditFailOnCritical(t *testing.T) {
	projDir, cfgPath, env := setupAuditProject(t)
	seedAdvisoryCache(t, env.CacheDir)

	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}

	code, out := runM(t, projDir, cfgPath, "audit", "--offline", "--fail-on", "critical")
	if code == 0 {
		t.Fatalf("expected nonzero exit with --fail-on critical, out=%s", out)
	}
	if !strings.Contains(out, "CVE-2026-0001") {
		t.Fatalf("expected findings in output before failure: %s", out)
	}
}

func TestAuditFailOnNonePassesWithFindings(t *testing.T) {
	projDir, cfgPath, env := setupAuditProject(t)
	seedAdvisoryCache(t, env.CacheDir)

	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}

	code, out := runM(t, projDir, cfgPath, "audit", "--offline", "--fail-on", "none")
	if code != 0 {
		t.Fatalf("audit exit=%d want 0 with --fail-on none, out=%s", code, out)
	}
	if !strings.Contains(out, "CVE-2026-0001") {
		t.Fatalf("missing CVE in output: %s", out)
	}
}

func TestAuditFailOnJSONEmitsBeforeExit(t *testing.T) {
	projDir, cfgPath, env := setupAuditProject(t)
	seedAdvisoryCache(t, env.CacheDir)

	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}

	code, out := runM(t, projDir, cfgPath, "audit", "--json", "--offline", "--fail-on", "critical")
	if code == 0 {
		t.Fatal("expected nonzero exit")
	}
	if !strings.Contains(out, `"schemaVersion"`) || !strings.Contains(out, `"CVE-2026-0001"`) {
		t.Fatalf("expected JSON report before failure exit: %s", out)
	}
}

func TestAuditOfflineFailsWithoutCache(t *testing.T) {
	projDir, cfgPath, _ := setupAuditProject(t)

	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}

	code, _ := runM(t, projDir, cfgPath, "audit", "--offline")
	if code == 0 {
		t.Fatal("expected non-zero exit without advisory cache in offline mode")
	}
}
