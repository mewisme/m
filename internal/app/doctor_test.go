package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
	"github.com/mewisme/mew/internal/transaction"
)

func setupDoctorHealthyProject(t *testing.T) (*Context, string) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")

	projDir := t.TempDir()
	fixture := filepath.Join(testkit.ModuleRoot(t), "fixtures", "projects", "mlock-greenfield")
	for _, name := range []string{"package.json", "m.lock"} {
		data, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(projDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: projDir, ConfigPath: cfgPath, Env: os.Environ()})
	if err != nil {
		t.Fatal(err)
	}
	return ac, projDir
}

func TestDoctorHealthyMlockProject(t *testing.T) {
	ac, _ := setupDoctorHealthyProject(t)
	report, err := Doctor(context.Background(), ac, DoctorOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Fatalf("expected ok report: %+v", report)
	}
	for _, id := range []string{"project", "config", "cache", "store", "lock", "filesystem", "transaction", "node"} {
		if findDoctorCheck(report, id) == nil {
			t.Fatalf("missing check %q: %+v", id, report.Checks)
		}
	}
	if findDoctorCheck(report, "project").Status != string(DoctorStatusOK) {
		t.Fatalf("project=%+v", findDoctorCheck(report, "project"))
	}
	if findDoctorCheck(report, "lock").Status != string(DoctorStatusOK) {
		t.Fatalf("lock=%+v", findDoctorCheck(report, "lock"))
	}
}

func TestDoctorStrictTreatsTxnWarnAsFailure(t *testing.T) {
	ac, projDir := setupDoctorHealthyProject(t)
	txnRoot := filepath.Join(projDir, ".mew", "txn", "orphan-test")
	if err := os.MkdirAll(filepath.Join(txnRoot, "stage"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := &transaction.Document{
		SchemaVersion: transaction.SchemaVersion,
		ID:            "orphan-test",
		ProjectRoot:   projDir,
		State:         transaction.StateStaging,
	}
	data, err := transaction.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(txnRoot, "journal.000001.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Doctor(context.Background(), ac, DoctorOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.OK {
		t.Fatal("expected strict failure with orphan txn")
	}
	txn := findDoctorCheck(report, "transaction")
	if txn == nil || txn.Status != string(DoctorStatusWarn) {
		t.Fatalf("transaction=%+v", txn)
	}
}

func TestDoctorFailsWithoutLock(t *testing.T) {
	ac, projDir := setupDoctorHealthyProject(t)
	if err := os.Remove(filepath.Join(projDir, "m.lock")); err != nil {
		t.Fatal(err)
	}

	report, err := Doctor(context.Background(), ac, DoctorOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.OK {
		t.Fatal("expected failure without lock")
	}
	lock := findDoctorCheck(report, "lock")
	if lock == nil || lock.Status != string(DoctorStatusFail) {
		t.Fatalf("lock=%+v", lock)
	}
}

func findDoctorCheck(rep DoctorReport, id string) *DoctorCheck {
	for i := range rep.Checks {
		if rep.Checks[i].ID == id {
			return &rep.Checks[i]
		}
	}
	return nil
}
