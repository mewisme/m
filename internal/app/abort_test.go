package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/fsx"
	"github.com/mewisme/m/internal/transaction"
)

func writeMinimalProject(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"abort-test","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func beginTestSession(t *testing.T, root string) (*MutationSession, context.Context) {
	t.Helper()
	ac := &Context{CWD: root, Config: &config.Effective{}}
	ctx := context.Background()
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	return sess, ctx
}

func TestAbortMutationFetchFailRollbackOK(t *testing.T) {
	root := t.TempDir()
	writeMinimalProject(t, root)
	sess, ctx := beginTestSession(t, root)
	txn := sess.Runner()

	liveLock := filepath.Join(root, "m.lock")
	if err := os.WriteFile(liveLock, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("m.lock"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(liveLock, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}

	fetchErr := apperr.New(apperr.Network, "app.install.fetch", "pkg", "fetch failed")
	res, err := abortMutation(ctx, sess, txn, fetchErr)
	if !errors.Is(err, fetchErr) {
		t.Fatalf("expected fetch err, got %v", err)
	}
	if apperr.CodeOf(err) != apperr.Network {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
	if !res.RolledBack {
		t.Fatal("expected RolledBack")
	}
	if res.RecoveryRequired {
		t.Fatal("unexpected RecoveryRequired")
	}
	data, readErr := os.ReadFile(liveLock)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "old" {
		t.Fatalf("lock not restored: %q", data)
	}
}

func TestAbortMutationBackupRestoreFail(t *testing.T) {
	root := t.TempDir()
	writeMinimalProject(t, root)
	sess, ctx := beginTestSession(t, root)
	txn := sess.Runner()

	liveLock := filepath.Join(root, "m.lock")
	if err := os.WriteFile(liveLock, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("m.lock"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(liveLock, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}

	restoreErr := errors.New("injected restore failure")
	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "rollback" && opIndex == 0 {
			return restoreErr
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	primary := apperr.New(apperr.Network, "app.install.fetch", "pkg", "fetch failed")
	res, err := abortMutation(ctx, sess, txn, primary)
	if !errors.Is(err, primary) {
		t.Fatalf("expected primary err, got %v", err)
	}
	if !errors.Is(err, restoreErr) {
		t.Fatalf("expected restore err in aggregate, got %v", err)
	}
	if res.RolledBack {
		t.Fatal("RolledBack should be false when backup restore fails")
	}
	if !res.RecoveryRequired {
		t.Fatal("expected RecoveryRequired")
	}
}

func TestAbortMutationCurrentCleanupFail(t *testing.T) {
	root := t.TempDir()
	writeMinimalProject(t, root)
	sess, ctx := beginTestSession(t, root)
	txn := sess.Runner()

	// Corrupt current metadata so clearCurrentVerified fails integrity check.
	txnDir := transaction.TxnRoot(root)
	if err := os.MkdirAll(txnDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(txnDir, "current.bad"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	primary := apperr.New(apperr.Install, "app.install.link", "", "link failed")
	res, err := abortMutation(ctx, sess, txn, primary)
	if !errors.Is(err, primary) {
		t.Fatalf("expected primary, got %v", err)
	}
	var of *apperr.OperationFailure
	if !errors.As(err, &of) {
		t.Fatal("expected OperationFailure")
	}
	if of.Cleanup == nil {
		t.Fatal("expected cleanup error in OperationFailure")
	}
	if apperr.CodeOf(of.Cleanup) != apperr.Integrity {
		t.Fatalf("cleanup code=%s", apperr.CodeOf(of.Cleanup))
	}
	if !res.TransactionCleanupIncomplete {
		t.Fatal("expected TransactionCleanupIncomplete")
	}
	found := false
	for _, code := range res.CleanupWarningCodes {
		if code == cleanupCodeTxnCurrentCleanup {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %s code, got %v", cleanupCodeTxnCurrentCleanup, res.CleanupWarningCodes)
	}
}

func TestAbortMutationLockReleaseFail(t *testing.T) {
	root := t.TempDir()
	writeMinimalProject(t, root)
	sess, ctx := beginTestSession(t, root)
	txn := sess.Runner()

	// Tamper lock owner so release fails with not-owner while lock metadata remains.
	lockDir := transaction.LockPath(root)
	ownerPath := filepath.Join(lockDir, fsx.OwnerFileName)
	data, err := os.ReadFile(ownerPath)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(data), txn.ID, "other-txn-id", 1)
	if tampered == string(data) {
		t.Fatal("failed to tamper lock owner txn id")
	}
	if err := os.WriteFile(ownerPath, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}

	primary := apperr.New(apperr.Transaction, "app.install.commit", "", "commit failed")
	res, err := abortMutation(ctx, sess, txn, primary)
	if !errors.Is(err, primary) {
		t.Fatalf("expected primary, got %v", err)
	}
	var of *apperr.OperationFailure
	if !errors.As(err, &of) || of.Cleanup == nil {
		t.Fatal("expected OperationFailure with cleanup")
	}
	if apperr.CodeOf(err) != apperr.Transaction {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
	if !res.TransactionCleanupIncomplete {
		t.Fatal("expected TransactionCleanupIncomplete")
	}
	found := false
	for _, code := range res.CleanupWarningCodes {
		if code == cleanupCodeTxnLockRelease {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s code, got %v", cleanupCodeTxnLockRelease, res.CleanupWarningCodes)
	}
}

func TestAbortMutationBothCleanupFailures(t *testing.T) {
	root := t.TempDir()
	writeMinimalProject(t, root)
	sess, ctx := beginTestSession(t, root)
	txn := sess.Runner()

	txnDir := transaction.TxnRoot(root)
	if err := os.MkdirAll(txnDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(txnDir, "current.bad"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	lockDir := transaction.LockPath(root)
	ownerPath := filepath.Join(lockDir, fsx.OwnerFileName)
	data, err := os.ReadFile(ownerPath)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(data), txn.ID, "other-txn-id", 1)
	if tampered == string(data) {
		t.Fatal("failed to tamper lock owner txn id")
	}
	if err := os.WriteFile(ownerPath, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}

	primary := apperr.New(apperr.Install, "app.install.link", "", "link failed")
	_, err = abortMutation(ctx, sess, txn, primary)
	if !errors.Is(err, primary) {
		t.Fatalf("expected primary, got %v", err)
	}
	var of *apperr.OperationFailure
	if !errors.As(err, &of) || of.Cleanup == nil {
		t.Fatal("expected OperationFailure with cleanup")
	}
	if apperr.CodeOf(of.Cleanup) == apperr.Internal {
		t.Fatalf("expected typed cleanup errors, got %v", of.Cleanup)
	}
	for _, code := range []apperr.Code{apperr.Integrity, apperr.Transaction} {
		found := false
		var walk func(error)
		walk = func(err error) {
			if err == nil || found {
				return
			}
			if apperr.CodeOf(err) == code {
				found = true
				return
			}
			switch e := err.(type) {
			case interface{ Unwrap() []error }:
				for _, child := range e.Unwrap() {
					walk(child)
				}
			default:
				walk(errors.Unwrap(err))
			}
		}
		walk(of.Cleanup)
		if !found {
			t.Fatalf("missing cleanup code %s in %v", code, of.Cleanup)
		}
	}
}

func TestAbortMutationOperationFailureShape(t *testing.T) {
	root := t.TempDir()
	writeMinimalProject(t, root)
	sess, ctx := beginTestSession(t, root)
	txn := sess.Runner()

	primary := apperr.New(apperr.Network, "app.install.fetch", "pkg", "fetch failed")
	_, err := abortMutation(ctx, sess, txn, primary)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, primary) {
		t.Fatalf("primary=%v", err)
	}
	if apperr.CodeOf(err) != apperr.Network {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestAbortMutationCleanupOnly(t *testing.T) {
	cleanup := errors.New("cleanup only")
	err := apperr.WithCleanup(nil, cleanup)
	if !errors.Is(err, cleanup) {
		t.Fatal("expected cleanup-only error")
	}
	var of *apperr.OperationFailure
	if errors.As(err, &of) && of.Primary != nil {
		t.Fatal("expected nil primary")
	}
}

func TestAbortMutationNoDoubleRollback(t *testing.T) {
	root := t.TempDir()
	writeMinimalProject(t, root)
	sess, ctx := beginTestSession(t, root)
	txn := sess.Runner()

	liveLock := filepath.Join(root, "m.lock")
	if err := os.WriteFile(liveLock, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	stage := txn.StagePath()
	if err := os.WriteFile(filepath.Join(stage, "m.lock"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("m.lock"); err != nil {
		t.Fatal(err)
	}
	if err := txn.SetPlan([]transaction.Op{{Kind: transaction.OpRename, Path: "m.lock", Backup: "stage/m.lock"}}); err != nil {
		t.Fatal(err)
	}

	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "commit" && opIndex == 0 {
			return os.ErrPermission
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	if err := txn.Commit(ctx, nil); err == nil {
		t.Fatal("expected commit failure")
	}

	rollbacks := 0
	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "rollback" {
			rollbacks++
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	primary := apperr.New(apperr.Transaction, "app.install.commit", "m.lock", "commit failed")
	_, err := abortMutation(ctx, sess, txn, primary)
	if err == nil {
		t.Fatal("expected error")
	}
	if rollbacks != 1 {
		t.Fatalf("expected single rollback, got %d", rollbacks)
	}
	data, readErr := os.ReadFile(liveLock)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "old" {
		t.Fatalf("lock not restored: %q", data)
	}
}

func TestAbortMutationRepeatedIdempotent(t *testing.T) {
	root := t.TempDir()
	writeMinimalProject(t, root)
	sess, ctx := beginTestSession(t, root)
	txn := sess.Runner()

	primary := apperr.New(apperr.Network, "app.install.fetch", "pkg", "fetch failed")
	res1, err1 := abortMutation(ctx, sess, txn, primary)
	if err1 == nil {
		t.Fatal("expected error")
	}
	res2, err2 := abortMutation(ctx, sess, nil, primary)
	if err2 == nil {
		t.Fatal("expected error on second abort")
	}
	if sess.runner != nil {
		t.Fatal("runner should be nil after first abort")
	}
	if res2.RolledBack {
		t.Fatal("second abort should not report RolledBack")
	}
	for _, code := range res2.CleanupWarningCodes {
		if code == cleanupCodeTxnLockRelease {
			t.Fatal("second abort should not attempt lock release")
		}
	}
	_ = res1
}

func TestMutationSessionAbortIdempotent(t *testing.T) {
	root := t.TempDir()
	writeMinimalProject(t, root)
	sess, ctx := beginTestSession(t, root)

	fr1, err1 := sess.Abort(ctx)
	if err1 != nil {
		t.Fatalf("first abort: %v", err1)
	}
	fr2, err2 := sess.Abort(ctx)
	if err2 != nil {
		t.Fatalf("second abort: %v", err2)
	}
	if fr2.LockReleaseRequested && fr2.LockReleaseResult == fsx.ReleaseNotOwner {
		t.Fatal("second abort must not report ReleaseNotOwner")
	}
	_ = fr1
}

func TestFormatInstallSummaryTransactionCleanup(t *testing.T) {
	summary := FormatInstallSummary(InstallResult{
		RolledBack:                   true,
		TransactionCleanupIncomplete: true,
		RecoveryRequired:             true,
		CleanupWarningCodes:          []string{cleanupCodeTxnLockRelease},
		CleanupWarnings:              []string{"lock release failed"},
	})
	if !strings.Contains(summary, "m recover") {
		t.Fatalf("missing recover hint: %q", summary)
	}
	if !strings.Contains(summary, "lock release failed") {
		t.Fatalf("missing warning: %q", summary)
	}

	committed := FormatInstallSummary(InstallResult{
		Committed:                    true,
		TransactionCleanupIncomplete: true,
		RecoveryRequired:             true,
		CleanupWarningCodes:          []string{cleanupCodeTxnCurrentCleanup},
		CleanupWarnings:              []string{"current cleanup failed"},
	})
	if !strings.Contains(committed, "Installation committed") {
		t.Fatalf("missing committed message: %q", committed)
	}
	if !strings.Contains(committed, "m recover") {
		t.Fatalf("missing recover hint: %q", committed)
	}

	ok := FormatInstallSummary(InstallResult{RolledBack: true})
	if strings.Contains(ok, "m recover") {
		t.Fatalf("clean rollback should not mention recover: %q", ok)
	}
}

func TestRecoverAfterAbortCleanupFailure(t *testing.T) {
	root := t.TempDir()
	writeMinimalProject(t, root)
	sess, ctx := beginTestSession(t, root)
	txn := sess.Runner()

	liveLock := filepath.Join(root, "m.lock")
	if err := os.WriteFile(liveLock, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("m.lock"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(liveLock, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := txn.SetState(transaction.StateCommitting); err != nil {
		t.Fatal(err)
	}

	primary := apperr.New(apperr.Transaction, "app.install.commit", "", "commit failed")
	if _, err := abortMutation(ctx, sess, txn, primary); err == nil {
		t.Fatal("expected abort error")
	}

	ac := &Context{CWD: root, Config: &config.Effective{}}
	result, err := Recover(ctx, ac)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "rolled_back" && result.Action != "discarded" && result.Action != "none" {
		t.Fatalf("unexpected action: %q", result.Action)
	}
	data, readErr := os.ReadFile(liveLock)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "old" {
		t.Fatalf("lock not restored after recover: %q", data)
	}
}

func TestPopulateAbortCleanupWarningOnlyFinishHook(t *testing.T) {
	var res InstallResult
	fr := transaction.FinishResult{
		CleanupWarnings:     []error{errors.New("finish hook failed")},
		CleanupWarningCodes: []string{"finish_hook"},
	}
	populateAbortCleanup(&res, fr)
	if res.TransactionCleanupIncomplete {
		t.Fatal("finish_hook must not set TransactionCleanupIncomplete")
	}
	if res.RecoveryRequired {
		t.Fatal("finish_hook must not set RecoveryRequired")
	}
	if !res.CleanupIncomplete {
		t.Fatal("expected CleanupIncomplete")
	}
}

func TestPopulateAbortCleanupWarningOnlyTxnDirRemove(t *testing.T) {
	var res InstallResult
	fr := transaction.FinishResult{
		CleanupWarnings:     []error{errors.New("txn dir remove failed")},
		CleanupWarningCodes: []string{"txn_dir_remove"},
	}
	populateAbortCleanup(&res, fr)
	if res.TransactionCleanupIncomplete {
		t.Fatal("txn_dir_remove must not set TransactionCleanupIncomplete")
	}
	if res.RecoveryRequired {
		t.Fatal("txn_dir_remove must not set RecoveryRequired")
	}
}

func TestPopulateAbortCleanupMixedSeverity(t *testing.T) {
	var res InstallResult
	fr := transaction.FinishResult{
		CleanupWarnings: []error{
			errors.New("finish hook failed"),
			errors.New("lock release failed"),
		},
		CleanupWarningCodes: []string{"finish_hook", cleanupCodeTxnLockRelease},
	}
	populateAbortCleanup(&res, fr)
	if !res.TransactionCleanupIncomplete {
		t.Fatal("expected TransactionCleanupIncomplete from critical code")
	}
	if !res.RecoveryRequired {
		t.Fatal("expected RecoveryRequired from critical code")
	}
	if len(res.CleanupWarningCodes) != 2 {
		t.Fatalf("codes=%v", res.CleanupWarningCodes)
	}
}

func TestPopulateAbortCleanupCriticalCurrentCleanup(t *testing.T) {
	var res InstallResult
	fr := transaction.FinishResult{
		CleanupWarnings:     []error{errors.New("current cleanup failed")},
		CleanupWarningCodes: []string{cleanupCodeTxnCurrentCleanup},
	}
	populateAbortCleanup(&res, fr)
	if !res.TransactionCleanupIncomplete {
		t.Fatal("expected TransactionCleanupIncomplete")
	}
	if !res.RecoveryRequired {
		t.Fatal("expected RecoveryRequired")
	}
}

func TestPopulateAbortCleanupRollbackFailPlusWarning(t *testing.T) {
	root := t.TempDir()
	writeMinimalProject(t, root)
	sess, ctx := beginTestSession(t, root)
	txn := sess.Runner()

	liveLock := filepath.Join(root, "m.lock")
	if err := os.WriteFile(liveLock, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("m.lock"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(liveLock, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}

	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "rollback" && opIndex == 0 {
			return errors.New("injected restore failure")
		}
		if phase == "finish" {
			return errors.New("finish hook failed")
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	primary := apperr.New(apperr.Network, "app.install.fetch", "pkg", "fetch failed")
	res, err := abortMutation(ctx, sess, txn, primary)
	if !errors.Is(err, primary) {
		t.Fatalf("expected primary, got %v", err)
	}
	if !res.RecoveryRequired {
		t.Fatal("rollback failure must set RecoveryRequired")
	}
	if apperr.CodeOf(err) != apperr.Network {
		t.Fatalf("primary code=%s", apperr.CodeOf(err))
	}
}
