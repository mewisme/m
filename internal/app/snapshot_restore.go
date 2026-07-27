package app

import (
	"context"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/lockfile/mlock"
	"github.com/mewisme/m/internal/snapshot"
)

// RestoreSnapshot restores manifest, lock, and node_modules in one transaction.
func RestoreSnapshot(ctx context.Context, ac *Context, id string) (InstallResult, error) {
	var res InstallResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return res, err
	}
	store := snapshot.NewStore(proj.Root)
	rec, err := store.Load(id)
	if err != nil {
		return res, err
	}
	lockDoc, err := mlock.Decode(rec.Lock)
	if err != nil {
		return res, err
	}
	g, err := mlock.ToGraph(lockDoc)
	if err != nil {
		return res, err
	}
	return runInstallTxn(ctx, ac, InstallOptions{
		Frozen:           true,
		WriteManifest:    true,
		PreResolvedGraph: g,
		StagedManifest:   rec.Manifest,
		StagedLock:       rec.Lock,
		SkipSnapshot:     true,
	}, nil)
}

// Rollback restores the previous snapshot (second-newest).
func Rollback(ctx context.Context, ac *Context) (InstallResult, error) {
	var res InstallResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return res, err
	}
	list, err := snapshot.NewStore(proj.Root).List()
	if err != nil {
		return res, err
	}
	if len(list) < 2 {
		return res, apperr.New(apperr.NotFound, "app.rollback", "", "no previous snapshot")
	}
	return RestoreSnapshot(ctx, ac, list[1].ID)
}
