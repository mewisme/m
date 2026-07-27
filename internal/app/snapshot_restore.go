package app

import (
	"context"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/snapshot"
)

// RestoreSnapshot restores manifest, lock, and node_modules in one transaction.
func RestoreSnapshot(ctx context.Context, ac *Context, id string) (InstallResult, error) {
	return restoreSnapshotByID(ctx, ac, id)
}

// Rollback restores the previous snapshot (second-newest) under mutation ownership.
func Rollback(ctx context.Context, ac *Context) (InstallResult, error) {
	var res InstallResult
	if ac == nil || ac.Config == nil {
		return res, apperr.New(apperr.Internal, "app.rollback", "", "missing app context")
	}
	root, err := resolveProjectRoot(ac, "")
	if err != nil {
		return res, err
	}
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		return res, err
	}
	list, err := snapshot.NewStore(root).List()
	if err != nil {
		return abortMutation(ctx, sess, sess.Runner(), err)
	}
	if len(list) < 2 {
		return abortMutation(ctx, sess, sess.Runner(), apperr.New(apperr.NotFound, "app.rollback", "", "no previous snapshot"))
	}
	return restoreSnapshotInSession(ctx, sess, list[1].ID)
}

func restoreSnapshotByID(ctx context.Context, ac *Context, id string) (InstallResult, error) {
	var res InstallResult
	if ac == nil || ac.Config == nil {
		return res, apperr.New(apperr.Internal, "app.restore", "", "missing app context")
	}
	root, err := resolveProjectRoot(ac, "")
	if err != nil {
		return res, err
	}
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		return res, err
	}
	return restoreSnapshotInSession(ctx, sess, id)
}

func restoreSnapshotInSession(ctx context.Context, sess *MutationSession, id string) (InstallResult, error) {
	var res InstallResult
	store := snapshot.NewStore(sess.projectRoot)
	rec, err := store.Load(id)
	if err != nil {
		return abortMutation(ctx, sess, sess.Runner(), err)
	}
	g, manifestBytes, err := snapshot.ValidateRestorePair(*rec)
	if err != nil {
		return abortMutation(ctx, sess, sess.Runner(), err)
	}
	res, err = runInstallInSession(ctx, sess, InstallOptions{
		Frozen:           true,
		WriteManifest:    true,
		PreResolvedGraph: g,
		StagedManifest:   manifestBytes,
		StagedLock:       rec.Lock,
		SkipSnapshot:     true,
	}, nil, nil)
	if err != nil {
		abortRes, abortErr := abortMutation(ctx, sess, sess.Runner(), err)
		res = mergeInstallResults(res, abortRes)
		return res, abortErr
	}
	finish, finishErr := sess.Finish(ctx, false)
	if finish.Committed {
		res.Committed = true
	}
	if finish.HasCriticalCleanupFailure() {
		populateCleanupResult(&res, finish)
		return res, finishErr
	}
	populateWarningCleanup(&res, finish)
	return res, finishErr
}
