package transaction_test

import (
	"errors"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fsx"
	"github.com/mewisme/m/internal/transaction"
)

func TestHasCriticalCleanupFailureReleaseNotOwner(t *testing.T) {
	fr := transaction.FinishResult{
		Committed:             true,
		CurrentCleared:        true,
		CurrentClearRequested: true,
		LockReleased:          true,
		LockReleaseRequested:  true,
		LockReleaseResult:     fsx.ReleaseNotOwner,
	}
	if !fr.HasCriticalCleanupFailure() {
		t.Fatal("ReleaseNotOwner must be a critical cleanup failure")
	}
}

func TestHasCriticalCleanupFailureLockReleased(t *testing.T) {
	fr := transaction.FinishResult{
		Committed:             true,
		CurrentCleared:        true,
		CurrentClearRequested: true,
		LockReleased:          true,
		LockReleaseRequested:  true,
	}
	if fr.HasCriticalCleanupFailure() {
		t.Fatal("expected success")
	}
}

func TestCleanupErrorCurrentCleanupFail(t *testing.T) {
	currentErr := apperr.New(apperr.Integrity, "transaction.current", "current.bad", "malformed current generation file")
	fr := transaction.FinishResult{
		CurrentClearRequested: true,
		CleanupWarnings:       []error{currentErr},
		CleanupWarningCodes:   []string{transaction.CleanupCodeTxnCurrentCleanup},
	}
	err := fr.CleanupError()
	if !errors.Is(err, currentErr) {
		t.Fatalf("expected current cleanup err, got %v", err)
	}
}

func TestCleanupErrorLockReleaseFail(t *testing.T) {
	lockErr := apperr.New(apperr.Transaction, "transaction.lock", "", "lock release failed")
	fr := transaction.FinishResult{
		LockReleaseRequested: true,
		LockReleaseResult:    fsx.ReleaseNotOwner,
		CleanupWarnings:      []error{lockErr},
		CleanupWarningCodes:  []string{transaction.CleanupCodeTxnLockRelease},
	}
	err := fr.CleanupError()
	if !errors.Is(err, lockErr) {
		t.Fatalf("expected lock err, got %v", err)
	}
}

func TestCleanupErrorBothFailures(t *testing.T) {
	currentErr := apperr.New(apperr.Integrity, "transaction.current", "current.bad", "malformed current generation file")
	lockErr := apperr.New(apperr.Transaction, "transaction.lock", "", "lock release failed")
	fr := transaction.FinishResult{
		CurrentClearRequested: true,
		LockReleaseRequested:  true,
		LockReleaseResult:     fsx.ReleaseNotOwner,
		CleanupWarnings:       []error{currentErr, lockErr},
		CleanupWarningCodes: []string{
			transaction.CleanupCodeTxnCurrentCleanup,
			transaction.CleanupCodeTxnLockRelease,
		},
	}
	err := fr.CleanupError()
	if !errors.Is(err, currentErr) {
		t.Fatalf("expected current cleanup err, got %v", err)
	}
	if !errors.Is(err, lockErr) {
		t.Fatalf("expected lock err, got %v", err)
	}
}

func TestCleanupErrorCriticalWithoutWarnings(t *testing.T) {
	fr := transaction.FinishResult{
		LockReleaseRequested: true,
		LockReleaseResult:    fsx.ReleaseNotOwner,
	}
	err := fr.CleanupError()
	if err == nil {
		t.Fatal("expected synthetic cleanup error")
	}
	if apperr.CodeOf(err) != apperr.Transaction {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestCleanupErrorTxnDirRemoveOnly(t *testing.T) {
	dirErr := apperr.New(apperr.IO, "transaction.finish", "txn", "remove failed")
	fr := transaction.FinishResult{
		Committed:           true,
		CleanupWarnings:     []error{dirErr},
		CleanupWarningCodes: []string{"txn_dir_remove"},
	}
	if fr.CleanupError() != nil {
		t.Fatalf("expected nil critical error, got %v", fr.CleanupError())
	}
	if fr.HasCriticalCleanupFailure() {
		t.Fatal("warning-only should not be critical")
	}
	warns := fr.WarningErrors()
	if len(warns) != 1 || !errors.Is(warns[0], dirErr) {
		t.Fatalf("warnings=%v", warns)
	}
}

func TestCleanupErrorFinishHookOnly(t *testing.T) {
	hookErr := apperr.New(apperr.Transaction, "transaction.finish", "", "hook failed")
	fr := transaction.FinishResult{
		Committed:           true,
		CleanupWarnings:     []error{hookErr},
		CleanupWarningCodes: []string{"finish_hook"},
	}
	if fr.CleanupError() != nil {
		t.Fatalf("expected nil critical error, got %v", fr.CleanupError())
	}
}

func TestCleanupErrorMixedCriticalAndWarning(t *testing.T) {
	currentErr := apperr.New(apperr.Integrity, "transaction.current", "current.bad", "malformed")
	dirErr := apperr.New(apperr.IO, "transaction.finish", "txn", "remove failed")
	fr := transaction.FinishResult{
		CurrentClearRequested: true,
		CleanupWarnings:       []error{dirErr, currentErr},
		CleanupWarningCodes:   []string{"txn_dir_remove", transaction.CleanupCodeTxnCurrentCleanup},
	}
	err := fr.CleanupError()
	if !errors.Is(err, currentErr) {
		t.Fatalf("expected current err, got %v", err)
	}
	if errors.Is(err, dirErr) {
		t.Fatal("dir remove should not be critical")
	}
}

func TestCleanupCodeSeverityUnknownDefaultsWarning(t *testing.T) {
	if transaction.CleanupCodeSeverity("unknown_future_code") != transaction.CleanupWarning {
		t.Fatal("unknown code should default to warning")
	}
}
