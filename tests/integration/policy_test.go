package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func setupPolicyDenyGPLProject(t *testing.T, policyFile string) (projDir, cfgPath string) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := testkit.LoadRegistry(t, "policy/deny-gpl/registry")
	srv := reg.Start(t)

	src := testkit.FixtureDir(t, "policy/deny-gpl")
	projDir = t.TempDir()
	testkit.CopyFixture(t, "policy/deny-gpl", projDir)

	policySrc := filepath.Join(src, policyFile)
	policyDst := filepath.Join(projDir, "mew.policy.json")
	data, err := os.ReadFile(policySrc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policyDst, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfgPath = filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return projDir, cfgPath
}

func TestPolicyInstallBlockedWithoutWaiver(t *testing.T) {
	projDir, cfgPath := setupPolicyDenyGPLProject(t, "mew.policy.json")
	code, _ := runM(t, projDir, cfgPath, "install")
	if code == 0 {
		t.Fatal("expected blocked install")
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "gpl-pkg")); err == nil {
		t.Fatal("gpl-pkg should not be installed when policy blocks")
	}
}

func TestPolicyCheckReportsGPLViolation(t *testing.T) {
	projDir, cfgPath := setupPolicyDenyGPLProject(t, "mew.policy.waiver.json")
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("install with waiver exit=%d out=%s", code, out)
	}

	denyPolicy, err := os.ReadFile(testkit.FixtureDir(t, "policy/deny-gpl/mew.policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "mew.policy.json"), denyPolicy, 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := runM(t, projDir, cfgPath, "policy", "check", "--json")
	if code == 0 {
		t.Fatalf("expected policy check failure, out=%s", out)
	}
	var doc struct {
		Passed     bool `json:"passed"`
		Violations []struct {
			Rule string `json:"rule"`
		} `json:"violations"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json: %v out=%s", err, out)
	}
	if doc.Passed || len(doc.Violations) == 0 || doc.Violations[0].Rule != "denied_license" {
		t.Fatalf("result=%+v out=%s", doc, out)
	}
}

func TestPolicyWaiverAllowsInstall(t *testing.T) {
	projDir, cfgPath := setupPolicyDenyGPLProject(t, "mew.policy.waiver.json")
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("install with waiver exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "gpl-pkg", "package.json")); err != nil {
		t.Fatal(err)
	}
	code, out = runM(t, projDir, cfgPath, "policy", "check", "--json")
	if code != 0 {
		t.Fatalf("policy check with waiver exit=%d out=%s", code, out)
	}
}
