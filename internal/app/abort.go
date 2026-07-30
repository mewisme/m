package app

import (
	"context"
	"errors"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
	"github.com/mewisme/mew/internal/transaction"
)

const (
	cleanupCodeTxnLockRelease         = "transaction_lock_release"
	cleanupCodeTxnCurrentCleanup      = "transaction_current_cleanup"
	cleanupCodeStoreImportLockRelease = "store_import_lock_release"
	cleanupCodeStoreIndexLockRelease  = "store_index_lock_release"
)

func abortMutation(ctx context.Context, sess *MutationSession, txn *transaction.Runner, primary error) (InstallResult, error) {
	var res InstallResult
	if primary == nil {
		return res, nil
	}
	var ac *Context
	if sess != nil {
		ac, _ = sess.AppContext()
	}
	phRollback := beginInstallPhase(ac, newInstallOpID(), phaseRollback)
	fr, cleanupErr, rolledBack := rollbackSession(ctx, sess, txn)
	res.RolledBack = rolledBack
	if !rolledBack {
		res.RecoveryRequired = true
		res.CleanupIncomplete = true
		if cleanupErr != nil {
			res.CleanupWarnings = append(res.CleanupWarnings, cleanupErr.Error())
		}
		phRollback.Complete(statusFailed)
	} else {
		phRollback.Complete(statusOK)
	}
	phCleanup := beginInstallPhase(ac, newInstallOpID(), phaseCleanup)
	populateAbortCleanup(&res, fr)
	if cleanupErr != nil || res.CleanupIncomplete {
		phCleanup.Complete(statusFailed)
	} else {
		phCleanup.Complete(statusOK)
	}
	return res, apperr.JoinCleanup(primary, cleanupErr)
}

// rollbackSession rolls back the active runner and releases the session-owned lock once.
func rollbackSession(ctx context.Context, sess *MutationSession, txn *transaction.Runner) (transaction.FinishResult, error, bool) {
	if txn == nil && sess != nil {
		txn = sess.runner
	}
	if txn == nil {
		return transaction.FinishResult{}, nil, false
	}
	fr, rollbackErr := txn.Rollback(ctx, transaction.DefaultFinishOpts())
	rolledBack := rollbackErr == nil
	var lockErr error
	if sess != nil && sess.runner != nil {
		txnID := sess.runner.ID
		sess.runner = nil
		lockErr = releaseSessionLock(sess.projectRoot, txnID, &fr)
	}
	return fr, joinSessionCleanup(fr, rollbackErr, lockErr), rolledBack
}

func joinSessionCleanup(fr transaction.FinishResult, rollbackErr, lockErr error) error {
	cleanupErr := fr.CriticalCleanupError()
	cleanupErr = joinDistinctCleanup(cleanupErr, rollbackErr)
	cleanupErr = joinDistinctCleanup(cleanupErr, lockErr)
	return cleanupErr
}

func joinDistinctCleanup(dst, err error) error {
	if err == nil {
		return dst
	}
	if dst == nil {
		return err
	}
	if errors.Is(dst, err) {
		return dst
	}
	return errors.Join(dst, err)
}

func releaseSessionLock(projectRoot, txnID string, fr *transaction.FinishResult) error {
	fr.LockReleaseRequested = true
	if err := transaction.ReleaseProjectLock(projectRoot, txnID); err != nil {
		fr.CleanupWarnings = append(fr.CleanupWarnings, err)
		fr.CleanupWarningCodes = append(fr.CleanupWarningCodes, cleanupCodeTxnLockRelease)
		fr.LockReleaseResult = fsx.ReleaseNotOwner
		return err
	}
	fr.LockReleased = true
	fr.LockReleaseResult = fsx.ReleaseOK
	return nil
}

func populateAbortCleanup(res *InstallResult, fr transaction.FinishResult) {
	if !fr.HasCriticalCleanupFailure() && len(fr.CleanupWarnings) == 0 {
		return
	}
	if fr.HasCriticalCleanupFailure() {
		res.CleanupIncomplete = true
		res.TransactionCleanupIncomplete = true
		res.RecoveryRequired = true
	}
	for i, w := range fr.CleanupWarnings {
		if w == nil {
			continue
		}
		code := ""
		if i < len(fr.CleanupWarningCodes) {
			code = fr.CleanupWarningCodes[i]
		}
		res.CleanupWarningCodes = append(res.CleanupWarningCodes, code)
		res.CleanupWarnings = append(res.CleanupWarnings, w.Error())
		if transaction.CleanupCodeSeverity(code) == transaction.CleanupCritical {
			res.CleanupIncomplete = true
			res.TransactionCleanupIncomplete = true
			res.RecoveryRequired = true
		} else {
			res.CleanupIncomplete = true
		}
	}
}
