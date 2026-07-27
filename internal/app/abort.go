package app

import (
	"context"
	"errors"

	"github.com/mewisme/m/internal/fsx"
	"github.com/mewisme/m/internal/transaction"
)

func abortMutation(ctx context.Context, sess *MutationSession, txn *transaction.Runner, primary error) (InstallResult, error) {
	var res InstallResult
	if primary == nil {
		return res, nil
	}
	if txn == nil && sess != nil {
		txn = sess.Runner()
	}
	var fr transaction.FinishResult
	var rollbackErr error
	if txn != nil {
		fr, rollbackErr = txn.Rollback(ctx, transaction.DefaultFinishOpts())
		res.RolledBack = rollbackErr == nil
		populateAbortCleanup(&res, fr)
	}
	if sess != nil && sess.runner != nil {
		txnID := sess.runner.ID
		sess.runner = nil
		if lockErr := releaseSessionLock(sess.projectRoot, txnID, &fr); lockErr != nil {
			rollbackErr = errors.Join(rollbackErr, lockErr)
		}
		populateAbortCleanup(&res, fr)
	}
	if fr.HasCriticalCleanupFailure() {
		res.RecoveryRequired = true
	}
	if rollbackErr != nil && !res.CleanupIncomplete {
		res.CleanupIncomplete = true
		res.CleanupWarnings = append(res.CleanupWarnings, rollbackErr.Error())
	}
	return res, primary
}

func releaseSessionLock(projectRoot, txnID string, fr *transaction.FinishResult) error {
	fr.LockReleaseRequested = true
	if err := transaction.ReleaseProjectLock(projectRoot, txnID); err != nil {
		fr.CleanupWarnings = append(fr.CleanupWarnings, err)
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
		res.RecoveryRequired = true
	}
	res.CleanupWarningCodes = append(res.CleanupWarningCodes, fr.CleanupWarningCodes...)
	for _, w := range fr.CleanupWarnings {
		if w != nil {
			res.CleanupWarnings = append(res.CleanupWarnings, w.Error())
		}
	}
}
