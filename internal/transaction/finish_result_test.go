package transaction_test

import (
	"testing"

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
