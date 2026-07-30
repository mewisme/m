package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/provenance"
	"github.com/mewisme/mew/internal/testkit"
)

func setProvenanceFixtureTrust(t *testing.T) {
	t.Helper()
	t.Setenv("MEW_PROVENANCE_TRUSTED_PUBLIC_KEY", provenance.FixturePublicKeyBase64())
}

func TestVerifyProvenanceAttestationPass(t *testing.T) {
	testkit.CleanEnv(t)
	setProvenanceFixtureTrust(t)
	projDir := t.TempDir()
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	att := filepath.Join(testkit.FixtureDir(t, "provenance/signed-pkg"), "signed-fixture-pkg-1.0.0.attestation.json")
	code, out := runM(t, projDir, cfgPath, "verify", "provenance", "--attestation", att)
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "Provenance verified") {
		t.Fatalf("stdout %q", out)
	}
}

func TestVerifyProvenanceWithLockPackagePass(t *testing.T) {
	testkit.CleanEnv(t)
	setProvenanceFixtureTrust(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "provenance/signed-pkg", projDir)
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "verify", "provenance", "signed-fixture-pkg@1.0.0")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "signed-fixture-pkg@1.0.0") {
		t.Fatalf("stdout %q", out)
	}
}

func TestVerifyProvenanceDigestMismatchFail(t *testing.T) {
	testkit.CleanEnv(t)
	setProvenanceFixtureTrust(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "provenance/signed-pkg", projDir)
	lockPath := filepath.Join(projDir, "m.lock")
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc mlock.Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Packages) == 0 {
		t.Fatal("expected package in lock")
	}
	doc.Packages[0].Integrity = "sha512-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="
	fixed, err := mlock.Encode(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, fixed, 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "verify", "provenance", "signed-fixture-pkg@1.0.0")
	if code == 0 {
		t.Fatalf("expected failure out=%s", out)
	}
}
