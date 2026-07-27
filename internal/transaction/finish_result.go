package transaction

import (
	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fsx"
)

// FinishResult reports post-mutation cleanup outcomes.
type FinishResult struct {
	Committed           bool
	LockReleased        bool
	CurrentCleared      bool
	LockReleaseResult   fsx.ReleaseResult
	CleanupWarnings     []error
	CleanupWarningCodes []string
}

func (fr FinishResult) HasCriticalCleanupFailure() bool {
	if fr.LockReleaseResult == fsx.ReleaseNotOwner {
		return true
	}
	return !fr.LockReleased || !fr.CurrentCleared
}

func appendCleanupWarning(fr *FinishResult, code string, err error) {
	if err == nil {
		return
	}
	fr.CleanupWarnings = append(fr.CleanupWarnings, err)
	if code != "" {
		fr.CleanupWarningCodes = append(fr.CleanupWarningCodes, code)
	}
}

func finishResultError(fr FinishResult) error {
	for _, err := range fr.CleanupWarnings {
		if err != nil {
			return err
		}
	}
	return apperr.New(apperr.Transaction, "transaction.cleanup", "", "critical cleanup incomplete")
}
