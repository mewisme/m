package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
)

func decodeSingleJSON(t *testing.T, out string) map[string]json.RawMessage {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(out))
	var doc map[string]json.RawMessage
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("json decode: %v out=%s", err, out)
	}
	if dec.More() {
		t.Fatalf("expected single JSON document, got trailing data: %s", out)
	}
	return doc
}

func TestWriteInstallResultJSONStoreMaintenance(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := app.InstallResult{
		Committed:                true,
		StoreMaintenanceRequired: true,
		StoreCleanupIncomplete:   true,
		CleanupWarningCodes:      []string{"store_import_lock_release"},
		CleanupWarnings:          []string{"lock not released"},
	}
	if err := writeInstallResult(cmd, result, true, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	doc := decodeSingleJSON(t, out)
	for _, key := range []string{
		"committed",
		"storeMaintenanceRequired",
		"storeCleanupIncomplete",
		"cleanupWarningCodes",
		"cleanupWarnings",
	} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("missing %q in %s", key, out)
		}
	}
	if strings.Contains(out, "m store status") {
		t.Fatalf("JSON output must not include prose hints: %s", out)
	}
	if strings.Contains(out, "m recover") {
		t.Fatalf("store-only should not suggest recover: %s", out)
	}
}

func TestWriteInstallResultJSONWarningOnlyFinish(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := app.InstallResult{
		Committed:           true,
		CleanupIncomplete:   true,
		CleanupWarningCodes: []string{"finish_hook"},
		CleanupWarnings:     []string{"finish hook failed"},
	}
	if err := writeInstallResult(cmd, result, true, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	doc := decodeSingleJSON(t, out)
	if _, ok := doc["recoveryRequired"]; ok {
		t.Fatalf("warning-only should not set recoveryRequired: %s", out)
	}
	if _, ok := doc["transactionCleanupIncomplete"]; ok {
		t.Fatalf("warning-only should not set transactionCleanupIncomplete: %s", out)
	}
	if !strings.Contains(out, "finish_hook") {
		t.Fatalf("missing warning code: %s", out)
	}
}

func TestWriteInstallResultJSONTransactionCleanup(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := app.InstallResult{
		Committed:                    true,
		TransactionCleanupIncomplete: true,
		CleanupIncomplete:            true,
		RecoveryRequired:             true,
		CleanupWarningCodes:          []string{"transaction_lock_release"},
		CleanupWarnings:              []string{"lock release failed"},
	}
	if err := writeInstallResult(cmd, result, true, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	doc := decodeSingleJSON(t, out)
	for _, key := range []string{
		"transactionCleanupIncomplete",
		"recoveryRequired",
		"cleanupIncomplete",
	} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("missing %q in %s", key, out)
		}
	}
	if strings.Contains(out, "m recover") {
		t.Fatalf("JSON output must not include prose hints: %s", out)
	}
}

func TestWriteInstallResultJSONAbortTransactionCleanup(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := app.InstallResult{
		RolledBack:                   true,
		TransactionCleanupIncomplete: true,
		CleanupIncomplete:            true,
		RecoveryRequired:             true,
		CleanupWarningCodes:          []string{"transaction_current_cleanup"},
		CleanupWarnings:              []string{"malformed current generation file"},
	}
	if err := writeInstallResult(cmd, result, true, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	doc := decodeSingleJSON(t, out)
	for _, key := range []string{
		"rolledBack",
		"transactionCleanupIncomplete",
		"cleanupIncomplete",
		"recoveryRequired",
		"cleanupWarningCodes",
		"cleanupWarnings",
	} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("missing %q in %s", key, out)
		}
	}
	if strings.Contains(out, "m recover") {
		t.Fatalf("JSON output must not include prose hints: %s", out)
	}
}

func TestWriteInstallResultJSONWarningOnlyCleanEOF(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := app.InstallResult{
		Committed:           true,
		CleanupIncomplete:   true,
		CleanupWarningCodes: []string{"txn_dir_remove"},
		CleanupWarnings:     []string{"txn dir remove failed"},
	}
	if err := writeInstallResult(cmd, result, true, false); err != nil {
		t.Fatal(err)
	}
	decodeSingleJSON(t, buf.String())
}

func TestWriteInstallResultJSONCombinedCriticalAndStore(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := app.InstallResult{
		Committed:                    true,
		TransactionCleanupIncomplete: true,
		RecoveryRequired:             true,
		StoreMaintenanceRequired:     true,
		CleanupWarningCodes:          []string{"transaction_lock_release", "store_import_lock_release"},
		CleanupWarnings:              []string{"lock release failed", "store lock not released"},
	}
	if err := writeInstallResult(cmd, result, true, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	doc := decodeSingleJSON(t, out)
	for _, key := range []string{
		"transactionCleanupIncomplete",
		"recoveryRequired",
		"storeMaintenanceRequired",
	} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("missing %q in %s", key, out)
		}
	}
	if strings.Contains(out, "m recover") || strings.Contains(out, "m store status") {
		t.Fatalf("JSON output must not include prose hints: %s", out)
	}
}

func TestWriteInstallResultSkipsSummaryOnRollback(t *testing.T) {
	cmd := &cobra.Command{}
	out := new(bytes.Buffer)
	errb := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(errb)
	err := writeInstallResult(cmd, app.InstallResult{Added: 1, Packages: 31, RolledBack: true}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("rollback must not print success summary on stdout: %q", out.String())
	}
	if !strings.Contains(errb.String(), "rolled back") {
		t.Fatalf("rollback framing missing on stderr: %q", errb.String())
	}
}

func TestWriteInstallResultPrintsAfterCommit(t *testing.T) {
	cmd := &cobra.Command{}
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	err := writeInstallResult(cmd, app.InstallResult{Added: 1, Packages: 31, Committed: true}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("Installed")) && !bytes.Contains(out.Bytes(), []byte("Added")) {
		t.Fatalf("committed install should print summary: %q", out.String())
	}
}

func TestWriteInstallResultHonorsNoSummary(t *testing.T) {
	root := &cobra.Command{Use: "m"}
	g := &globalFlags{noSummary: true}
	flagOwners.Store(root, g)
	cmd := &cobra.Command{Use: "install"}
	root.AddCommand(cmd)
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	err := writeInstallResult(cmd, app.InstallResult{Added: 1, Packages: 31, Committed: true}, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("--no-summary must suppress success summary: %q", out.String())
	}
}

func TestWriteInstallResultDryRun(t *testing.T) {
	cmd := &cobra.Command{}
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	err := writeInstallResult(cmd, app.InstallResult{Added: 2, Packages: 2}, false, true)
	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Planned changes") {
		t.Fatalf("%q", got)
	}
	if !strings.Contains(got, "No project files were changed") {
		t.Fatalf("%q", got)
	}
}

func TestWriteInstallResultJSONStillPrintsOnRollback(t *testing.T) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	result := app.InstallResult{RolledBack: true, Added: 1, Packages: 31}
	if err := writeInstallResult(cmd, result, true, false); err != nil {
		t.Fatal(err)
	}
	doc := decodeSingleJSON(t, buf.String())
	if _, ok := doc["rolledBack"]; !ok {
		t.Fatalf("json rollback result should still encode: %s", buf.String())
	}
}
