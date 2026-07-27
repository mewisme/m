package app

import (
	"context"
	"errors"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fsx"
	"github.com/mewisme/m/internal/transaction"
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
	fr, rollbackErr, rolledBack := rollbackSession(ctx, sess, txn)
	res.RolledBack = rolledBack
	if !rolledBack {
		res.RecoveryRequired = true
		res.CleanupIncomplete = true
		if rollbackErr != nil {
			res.CleanupWarnings = append(res.CleanupWarnings, rollbackErr.Error())
		}
	}
	populateAbortCleanup(&res, fr)
	return res, apperr.JoinCleanup(primary, rollbackErr)
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
	var cleanupErr error
	if rollbackErr != nil {
		cleanupErr = rollbackErr
	}
	if sess != nil && sess.runner != nil {
		txnID := sess.runner.ID
		sess.runner = nil
		if lockErr := releaseSessionLock(sess.projectRoot, txnID, &fr); lockErr != nil {
			cleanupErr = errors.Join(cleanupErr, lockErr)
		}
	}
	return fr, cleanupErr, rolledBack
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
	} else if len(fr.CleanupWarnings) > 0 {
		res.CleanupIncomplete = true
		res.TransactionCleanupIncomplete = true
	}
	res.CleanupWarningCodes = append(res.CleanupWarningCodes, fr.CleanupWarningCodes...)
	for _, w := range fr.CleanupWarnings {
		if w != nil {
			res.CleanupWarnings = append(res.CleanupWarnings, w.Error())
		}
	}
}
