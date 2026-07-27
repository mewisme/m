package transaction

import (
	"errors"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fsx"
)

// FinishResult reports post-mutation cleanup outcomes.
type FinishResult struct {
	Committed             bool
	LockReleased          bool
	CurrentCleared        bool
	LockReleaseRequested  bool
	CurrentClearRequested bool
	LockReleaseResult     fsx.ReleaseResult
	CleanupWarnings       []error
	CleanupWarningCodes   []string
}

func (fr FinishResult) HasCriticalCleanupFailure() bool {
	if fr.LockReleaseRequested && fr.LockReleaseResult == fsx.ReleaseNotOwner {
		return true
	}
	if fr.LockReleaseRequested && !fr.LockReleased {
		return true
	}
	if fr.CurrentClearRequested && !fr.CurrentCleared {
		return true
	}
	for _, code := range fr.CleanupWarningCodes {
		if CleanupCodeSeverity(code) == CleanupCritical {
			return true
		}
	}
	return false
}

const (
	CleanupCodeTxnLockRelease    = "transaction_lock_release"
	CleanupCodeTxnCurrentCleanup = "transaction_current_cleanup"
)

func appendCleanupWarning(fr *FinishResult, code string, err error) {
	if err == nil {
		return
	}
	fr.CleanupWarnings = append(fr.CleanupWarnings, err)
	if code != "" {
		fr.CleanupWarningCodes = append(fr.CleanupWarningCodes, code)
	}
}

// CriticalCleanupError joins critical cleanup warnings and reports structural cleanup failures.
func (fr FinishResult) CriticalCleanupError() error {
	var errs []error
	for i, err := range fr.CleanupWarnings {
		if err == nil {
			continue
		}
		code := ""
		if i < len(fr.CleanupWarningCodes) {
			code = fr.CleanupWarningCodes[i]
		}
		if CleanupCodeSeverity(code) == CleanupCritical {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	if fr.LockReleaseRequested && fr.LockReleaseResult == fsx.ReleaseNotOwner {
		return apperr.New(apperr.Transaction, "transaction.cleanup", "", "critical cleanup incomplete")
	}
	if fr.LockReleaseRequested && !fr.LockReleased {
		return apperr.New(apperr.Transaction, "transaction.cleanup", "", "critical cleanup incomplete")
	}
	if fr.CurrentClearRequested && !fr.CurrentCleared {
		return apperr.New(apperr.Transaction, "transaction.cleanup", "", "critical cleanup incomplete")
	}
	return nil
}

// WarningErrors returns non-critical cleanup warnings only.
func (fr FinishResult) WarningErrors() []error {
	var out []error
	for i, err := range fr.CleanupWarnings {
		if err == nil {
			continue
		}
		code := ""
		if i < len(fr.CleanupWarningCodes) {
			code = fr.CleanupWarningCodes[i]
		}
		if CleanupCodeSeverity(code) == CleanupWarning {
			out = append(out, err)
		}
	}
	return out
}

// CleanupError reports critical cleanup failures only (non-critical warnings are excluded).
func (fr FinishResult) CleanupError() error {
	return fr.CriticalCleanupError()
}

func finishResultError(fr FinishResult) error {
	return fr.CriticalCleanupError()
}
