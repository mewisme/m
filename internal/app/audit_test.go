package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/advisory"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/testkit"
)

func TestAuditReportsVulnerability(t *testing.T) {
	env := testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	reg := testkit.LoadRegistry(t, "registry/audit/v1")
	srv := reg.Start(t)

	projDir := t.TempDir()
	fixture := filepath.Join(testkit.ModuleRoot(t), "fixtures", "audit", "vulnerable-transitive", "package.json")
	pkgJSON, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), pkgJSON, 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: projDir, ConfigPath: cfgPath, Env: os.Environ()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Install(ctx, ac, InstallOptions{}); err != nil {
		t.Fatal(err)
	}

	store := advisory.Store{Dir: filepath.Join(env.CacheDir, "advisory")}
	if err := store.SeedFixture(testkit.ModuleRoot(t)); err != nil {
		t.Fatal(err)
	}

	result, err := Audit(ctx, ac, AuditOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Report.Vulnerabilities) != 1 {
		t.Fatalf("vulns=%d want 1: %+v", len(result.Report.Vulnerabilities), result.Report.Vulnerabilities)
	}
	v := result.Report.Vulnerabilities[0]
	if v.ID != "CVE-2026-0001" || v.Package != "vuln-pkg" || v.Version != "1.0.0" {
		t.Fatalf("unexpected vuln: %+v", v)
	}
}

func TestAuditFixSuggestsSafeVersion(t *testing.T) {
	env := testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	reg := testkit.LoadRegistry(t, "registry/audit/v1")
	srv := reg.Start(t)

	projDir := t.TempDir()
	fixture := filepath.Join(testkit.ModuleRoot(t), "fixtures", "audit", "vulnerable-transitive", "package.json")
	pkgJSON, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), pkgJSON, 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: projDir, ConfigPath: cfgPath, Env: os.Environ()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Install(ctx, ac, InstallOptions{}); err != nil {
		t.Fatal(err)
	}

	store := advisory.Store{Dir: filepath.Join(env.CacheDir, "advisory")}
	if err := store.SeedFixture(testkit.ModuleRoot(t)); err != nil {
		t.Fatal(err)
	}

	result, err := Audit(ctx, ac, AuditOptions{Fix: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Fixes) != 1 {
		t.Fatalf("fixes=%d want 1: %+v", len(result.Fixes), result.Fixes)
	}
	if result.Fixes[0].Package != "vuln-pkg" || result.Fixes[0].ToVersion != "1.0.1" {
		t.Fatalf("fix=%+v", result.Fixes[0])
	}
	_ = env
}

func TestAuditOfflineMissingCache(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	reg := testkit.LoadRegistry(t, "registry/audit/v1")
	srv := reg.Start(t)

	projDir := t.TempDir()
	fixture := filepath.Join(testkit.ModuleRoot(t), "fixtures", "audit", "vulnerable-transitive", "package.json")
	pkgJSON, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), pkgJSON, 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: projDir, ConfigPath: cfgPath, Env: os.Environ()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Install(ctx, ac, InstallOptions{}); err != nil {
		t.Fatal(err)
	}

	acOffline, err := New(ctx, Options{CWD: projDir, ConfigPath: cfgPath, Env: os.Environ(), Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Audit(ctx, acOffline, AuditOptions{})
	if err == nil {
		t.Fatal("expected error without advisory cache in offline mode")
	}
	if apperr.CodeOf(err) != apperr.Network {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}
