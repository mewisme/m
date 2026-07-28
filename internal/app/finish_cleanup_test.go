package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/testkit"
	"github.com/mewisme/mew/internal/transaction"
)

func TestFinishWarningOnlyTxnDirRemove(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	defer srv.Close()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{
  "name": "warn-only",
  "version": "1.0.0"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(root, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ac, err := New(ctx, Options{CWD: root, ConfigPath: cfgPath})
	if err != nil {
		t.Fatal(err)
	}

	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "finish" {
			return errors.New("injected finish hook failure")
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	res, err := Add(ctx, ac, "lodash", AddOptions{})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if !res.Committed {
		t.Fatal("expected committed install")
	}
	if res.RecoveryRequired {
		t.Fatal("warning-only finish must not require recovery")
	}
	if len(res.CleanupWarnings) == 0 {
		t.Fatal("expected cleanup warnings in result")
	}
	found := false
	for _, code := range res.CleanupWarningCodes {
		if code == "finish_hook" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("codes=%v", res.CleanupWarningCodes)
	}
}

func TestFinishCriticalCurrentCleanupSetsRecovery(t *testing.T) {
	root := t.TempDir()
	writeMinimalProject(t, root)
	sess, ctx := beginTestSession(t, root)

	txnDir := transaction.TxnRoot(root)
	if err := os.MkdirAll(txnDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(txnDir, "current.bad"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	fr := txnFinishWithCurrentCleanup(sess, ctx)
	if !fr.HasCriticalCleanupFailure() {
		t.Fatal("expected critical cleanup failure")
	}
	var res InstallResult
	populateCleanupResult(&res, fr)
	if !res.RecoveryRequired {
		t.Fatal("expected RecoveryRequired")
	}
	if !res.TransactionCleanupIncomplete {
		t.Fatal("expected TransactionCleanupIncomplete")
	}
}

func txnFinishWithCurrentCleanup(sess *MutationSession, ctx context.Context) transaction.FinishResult {
	if sess == nil || sess.runner == nil {
		return transaction.FinishResult{}
	}
	txnID := sess.runner.ID
	fr := sess.runner.Finish(false, transaction.FinishOpts{ClearCurrent: true})
	_ = releaseSessionLock(sess.projectRoot, txnID, &fr)
	sess.runner = nil
	return fr
}

func TestPopulateWarningCleanupNoRecovery(t *testing.T) {
	hookErr := errors.New("housekeeping failed")
	fr := transaction.FinishResult{
		Committed:           true,
		CleanupWarnings:     []error{hookErr},
		CleanupWarningCodes: []string{"finish_hook"},
	}
	var res InstallResult
	populateWarningCleanup(&res, fr)
	if res.RecoveryRequired {
		t.Fatal("warnings must not set RecoveryRequired")
	}
	if !res.CleanupIncomplete {
		t.Fatal("expected CleanupIncomplete")
	}
	if res.TransactionCleanupIncomplete {
		t.Fatal("warning-only must not set TransactionCleanupIncomplete")
	}
}

// compile-time guard that config package is linked for finish tests using registry fixtures.
var _ = config.SourceCLI

func TestFormatInstallSummaryWarningOnlyCommitted(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:           true,
		CleanupIncomplete:   true,
		CleanupWarningCodes: []string{"finish_hook"},
		CleanupWarnings:     []string{"finish hook failed"},
	})
	if strings.Contains(summary, "m recover") {
		t.Fatalf("warning-only must not suggest recover: %q", summary)
	}
	if !strings.Contains(summary, "non-critical cleanup warning") {
		t.Fatalf("missing warning message: %q", summary)
	}
	if !strings.Contains(summary, "finish hook failed") {
		t.Fatalf("missing warning detail: %q", summary)
	}
}

func TestFormatInstallSummaryCriticalCommitted(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:                    true,
		TransactionCleanupIncomplete: true,
		RecoveryRequired:             true,
		CleanupWarningCodes:          []string{cleanupCodeTxnLockRelease},
		CleanupWarnings:              []string{"lock release failed"},
	})
	if !strings.Contains(summary, "m recover") {
		t.Fatalf("critical must suggest recover: %q", summary)
	}
}

func TestFormatInstallSummaryStoreMaintenance(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:                true,
		StoreMaintenanceRequired: true,
		CleanupWarningCodes:      []string{cleanupCodeStoreImportLockRelease},
		CleanupWarnings:          []string{"store lock not released"},
	})
	if !strings.Contains(summary, "m store status") {
		t.Fatalf("missing store hint: %q", summary)
	}
	if strings.Contains(summary, "m recover") {
		t.Fatalf("store-only must not suggest recover: %q", summary)
	}
}

func TestFormatInstallSummaryFinishHookOnly(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:           true,
		CleanupIncomplete:   true,
		CleanupWarningCodes: []string{"finish_hook"},
		CleanupWarnings:     []string{"finish hook failed"},
	})
	if strings.Count(summary, "finish hook failed") != 1 {
		t.Fatalf("expected one finish_hook line: %q", summary)
	}
	if strings.Contains(summary, "m store status") {
		t.Fatalf("finish_hook only must not show store block: %q", summary)
	}
}

func TestFormatInstallSummaryTxnDirRemoveOnly(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:           true,
		CleanupIncomplete:   true,
		CleanupWarningCodes: []string{"txn_dir_remove"},
		CleanupWarnings:     []string{"txn dir remove failed"},
	})
	if strings.Count(summary, "txn dir remove failed") != 1 {
		t.Fatalf("expected one txn_dir_remove line: %q", summary)
	}
}

func TestFormatInstallSummaryCombinedTxnAndStore(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:                true,
		CleanupIncomplete:        true,
		StoreMaintenanceRequired: true,
		CleanupWarningCodes: []string{
			"finish_hook",
			cleanupCodeStoreImportLockRelease,
		},
		CleanupWarnings: []string{
			"finish hook failed",
			"store lock not released",
		},
	})
	if strings.Count(summary, "finish hook failed") != 1 {
		t.Fatalf("missing txn warning: %q", summary)
	}
	if strings.Count(summary, "store lock not released") != 1 {
		t.Fatalf("missing store warning: %q", summary)
	}
	if strings.Contains(summary, "m recover") {
		t.Fatalf("non-critical txn must not suggest recover: %q", summary)
	}
}

func TestFormatInstallSummaryCriticalPlusStore(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:                    true,
		TransactionCleanupIncomplete: true,
		RecoveryRequired:             true,
		StoreMaintenanceRequired:     true,
		CleanupWarningCodes: []string{
			cleanupCodeTxnLockRelease,
			cleanupCodeStoreImportLockRelease,
		},
		CleanupWarnings: []string{
			"lock release failed",
			"store lock not released",
		},
	})
	if strings.Count(summary, "lock release failed") != 1 {
		t.Fatalf("missing critical txn warning: %q", summary)
	}
	if strings.Count(summary, "store lock not released") != 1 {
		t.Fatalf("missing store warning: %q", summary)
	}
	if !strings.Contains(summary, "m recover") {
		t.Fatalf("critical must suggest recover: %q", summary)
	}
}

func TestFormatInstallSummaryCriticalPlusFinishHook(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:                    true,
		CleanupIncomplete:            true,
		TransactionCleanupIncomplete: true,
		RecoveryRequired:             true,
		CleanupWarningCodes: []string{
			cleanupCodeTxnLockRelease,
			"finish_hook",
		},
		CleanupWarnings: []string{
			"lock release failed",
			"finish hook failed",
		},
	})
	if !strings.Contains(summary, "m recover") {
		t.Fatalf("critical must suggest recover: %q", summary)
	}
	if !strings.Contains(summary, "non-critical cleanup warning") {
		t.Fatalf("missing non-critical header: %q", summary)
	}
	if strings.Count(summary, "lock release failed") != 1 {
		t.Fatalf("critical detail: %q", summary)
	}
	if strings.Count(summary, "finish hook failed") != 1 {
		t.Fatalf("non-critical detail: %q", summary)
	}
}

func TestFormatInstallSummaryCriticalPlusTxnDirRemove(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:                    true,
		CleanupIncomplete:            true,
		TransactionCleanupIncomplete: true,
		RecoveryRequired:             true,
		CleanupWarningCodes: []string{
			cleanupCodeTxnCurrentCleanup,
			"txn_dir_remove",
		},
		CleanupWarnings: []string{
			"current cleanup failed",
			"txn dir remove failed",
		},
	})
	if strings.Count(summary, "current cleanup failed") != 1 {
		t.Fatalf("critical detail: %q", summary)
	}
	if strings.Count(summary, "txn dir remove failed") != 1 {
		t.Fatalf("non-critical detail: %q", summary)
	}
}

func TestFormatInstallSummaryCriticalNonCriticalAndStore(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:                    true,
		CleanupIncomplete:            true,
		TransactionCleanupIncomplete: true,
		RecoveryRequired:             true,
		StoreMaintenanceRequired:     true,
		CleanupWarningCodes: []string{
			cleanupCodeTxnLockRelease,
			"finish_hook",
			cleanupCodeStoreImportLockRelease,
		},
		CleanupWarnings: []string{
			"lock release failed",
			"finish hook failed",
			"store lock not released",
		},
	})
	if !strings.Contains(summary, "m recover") {
		t.Fatalf("critical must suggest recover: %q", summary)
	}
	if !strings.Contains(summary, "non-critical cleanup warning") {
		t.Fatalf("missing non-critical header: %q", summary)
	}
	if !strings.Contains(summary, "m store status") {
		t.Fatalf("missing store hint: %q", summary)
	}
	if strings.Count(summary, "lock release failed") != 1 ||
		strings.Count(summary, "finish hook failed") != 1 ||
		strings.Count(summary, "store lock not released") != 1 {
		t.Fatalf("all sections once: %q", summary)
	}
}

func TestFormatInstallSummaryDedupesDuplicateWarnings(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:         true,
		CleanupIncomplete: true,
		CleanupWarningCodes: []string{
			"finish_hook",
			"finish_hook",
		},
		CleanupWarnings: []string{
			"finish hook failed",
			"finish hook failed",
		},
	})
	if strings.Count(summary, "finish hook failed") != 1 {
		t.Fatalf("duplicate warnings should print once: %q", summary)
	}
}

func TestFormatInstallSummarySameCodeDifferentMessage(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:         true,
		CleanupIncomplete: true,
		CleanupWarningCodes: []string{
			"finish_hook",
			"finish_hook",
		},
		CleanupWarnings: []string{
			"first failure",
			"second failure",
		},
	})
	if strings.Count(summary, "first failure") != 1 {
		t.Fatalf("missing first failure: %q", summary)
	}
	if strings.Count(summary, "second failure") != 1 {
		t.Fatalf("missing second failure: %q", summary)
	}
}

func TestFormatInstallSummaryUnknownCode(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		Committed:           true,
		CleanupIncomplete:   true,
		CleanupWarningCodes: []string{"custom_warning"},
		CleanupWarnings:     []string{"something odd happened"},
	})
	if strings.Count(summary, "something odd happened") != 1 {
		t.Fatalf("unknown code should appear once: %q", summary)
	}
	if strings.Contains(summary, "m store status") {
		t.Fatalf("unknown txn code must not trigger store block: %q", summary)
	}
}
