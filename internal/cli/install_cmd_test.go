package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/app"
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
