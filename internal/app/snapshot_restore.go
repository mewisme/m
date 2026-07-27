package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/snapshot"
	"github.com/mewisme/m/internal/transaction"
)

// RestoreSnapshot copies snapshot manifest+lock to live and reinstalls frozen from cache.
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
	txn := transaction.NewRunner(proj.Root)
	if err := txn.Begin(ctx); err != nil {
		return res, err
	}
	stage := txn.StagePath()
	if err := os.WriteFile(filepath.Join(stage, "package.json"), rec.Manifest, 0o644); err != nil {
		_ = txn.Rollback(ctx)
		return res, apperr.Wrap(apperr.IO, "app.restore", "package.json", err)
	}
	if err := os.WriteFile(filepath.Join(stage, lockFileName), rec.Lock, 0o644); err != nil {
		_ = txn.Rollback(ctx)
		return res, apperr.Wrap(apperr.IO, "app.restore", lockFileName, err)
	}
	plan := []transaction.Op{
		{Kind: transaction.OpRename, Path: "package.json", Backup: "stage/package.json"},
		{Kind: transaction.OpRename, Path: lockFileName, Backup: "stage/" + lockFileName},
	}
	if err := txn.SetPlan(plan); err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}
	for _, rel := range []string{"package.json", lockFileName} {
		if err := txn.RecordBackup(rel); err != nil {
			_ = txn.Rollback(ctx)
			return res, err
		}
	}
	if err := txn.Commit(ctx, nil); err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}
	manifest.Invalidate(proj.Root)
	_ = txn.Finish(false)
	return Install(ctx, ac, InstallOptions{Frozen: true})
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
