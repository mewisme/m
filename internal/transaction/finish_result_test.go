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
