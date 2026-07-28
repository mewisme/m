package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/transaction"
)

// MigrateLockOptions controls explicit lock migration to m.lock.
type MigrateLockOptions struct {
	From      string // nub|pnpm; empty uses project identity
	To        string // m
	DryRun    bool
	PnpmMajor int
}

// MigrateLockResult summarizes migrate output.
type MigrateLockResult struct {
	DryRun     bool
	LossReport lockfile.LossReport
	Path       string
}

// LockDiffOptions compares incumbent lock graph to another lock path.
type LockDiffOptions struct {
	OtherPath string
	PnpmMajor int
}

// LockDiff compares the incumbent lock graph to another lockfile.
func LockDiff(ctx context.Context, ac *Context, opts LockDiffOptions) (*lockfile.GraphDiff, error) {
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return nil, err
	}
	left, err := lockfile.ReadGraph(ctx, proj.Root, proj.Identity)
	if err != nil {
		return nil, err
	}
	otherPath := opts.OtherPath
	if otherPath == "" {
		otherPath = filepath.Join(proj.Root, "m.lock")
	}
	otherID, ok := lockIdentityFromBasename(filepath.Base(otherPath))
	if !ok {
		return nil, apperr.New(apperr.Usage, "lock.diff", otherPath, "unrecognized lockfile name")
	}
	adapter := lockfile.AdapterFor(otherID)
	if adapter == nil {
		return nil, lockfile.NewUnsupported("lock.diff", otherPath, "adapter not registered")
	}
	right, err := adapter.Read(ctx, otherPath)
	if err != nil {
		return nil, err
	}
	return lockfile.DiffGraphs(left, right)
}

// MigrateLock migrates an incumbent nub/pnpm lock to m.lock.
func MigrateLock(ctx context.Context, ac *Context, opts MigrateLockOptions) (MigrateLockResult, error) {
	var out MigrateLockResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return out, err
	}
	fromID, err := resolveMigrateFrom(proj, opts.From)
	if err != nil {
		return out, err
	}
	to := opts.To
	if to == "" {
		to = "m"
	}
	if to != "m" {
		return out, apperr.New(apperr.Usage, "lock.migrate", to, "only --to m is supported")
	}
	lockPath, ok := project.IncumbentLockPath(proj.Root, fromID)
	if !ok {
		return out, apperr.New(apperr.NotFound, "lock.migrate", project.LockFilename(fromID), "source lock not found")
	}
	prior, err := os.ReadFile(lockPath)
	if err != nil {
		return out, apperr.Wrap(apperr.IO, "lock.migrate", lockPath, err)
	}
	ext, ok := lockfile.ExtAdapterFor(fromID)
	if !ok {
		return out, lockfile.NewUnsupported("lock.migrate", project.LockFilename(fromID), "adapter not registered")
	}
	g, _, err := ext.ReadWithExtensions(ctx, lockPath)
	if err != nil {
		return out, err
	}
	loss, err := ext.LossFromDocument(ctx, prior)
	if err != nil {
		return out, err
	}
	mext, ok := lockfile.ExtAdapterFor(project.IdentityMew)
	if !ok {
		return out, lockfile.NewUnsupported("lock.migrate", "m.lock", "m.lock adapter not registered")
	}
	det := lockfile.Detection{Confidence: lockfile.DetectionCertain}
	if fromID == project.IdentityPNPM {
		det, err = lockfile.DetectPnpmWithMajor(prior, opts.PnpmMajor)
		if err != nil {
			return out, err
		}
		if opts.PnpmMajor != 0 {
			det.ExplicitMajor = true
		}
	}
	encodeRes, encErr := lockfile.EncodePreserving(ctx, mext, filepath.Join(proj.Root, "m.lock"), g, nil, nil, det)
	if encErr != nil {
		var rep *lockfile.RepresentabilityError
		if errors.As(encErr, &rep) {
			loss.Items = append(loss.Items, rep.Report.Items...)
		} else if !opts.DryRun {
			return out, encErr
		}
	}
	_ = loss.Normalize()
	out.LossReport = loss
	out.Path = filepath.Join(proj.Root, "m.lock")
	if opts.DryRun {
		out.DryRun = true
		return out, nil
	}
	if encErr != nil {
		return out, encErr
	}
	if len(loss.Items) > 0 {
		return out, lockfile.NewUnrepresentable("lock.migrate", "m.lock", "lossy migration", loss)
	}
	if err := commitMigratedLock(ctx, ac, proj, encodeRes.Bytes); err != nil {
		return out, err
	}
	return out, nil
}

func commitMigratedLock(ctx context.Context, ac *Context, proj *project.Project, data []byte) error {
	sess, err := BeginMutationSession(ctx, ac, proj.Root)
	if err != nil {
		return err
	}
	txn := sess.Runner()
	stage := txn.StagePath()
	stageLock := filepath.Join(stage, "m.lock")
	if err := os.WriteFile(stageLock, data, 0o644); err != nil {
		_, _ = sess.Abort(ctx)
		return apperr.Wrap(apperr.IO, "lock.migrate", stageLock, err)
	}
	plan := []transaction.Op{{Kind: transaction.OpRename, Path: "m.lock", Backup: filepath.Join("stage", "m.lock")}}
	if err := txn.SetPlan(plan); err != nil {
		_, _ = sess.Abort(ctx)
		return err
	}
	if _, err := os.Stat(filepath.Join(proj.Root, "m.lock")); err == nil {
		if err := txn.RecordBackup("m.lock"); err != nil {
			_, _ = sess.Abort(ctx)
			return err
		}
	}
	if err := txn.Commit(ctx, nil); err != nil {
		_, _ = sess.Abort(ctx)
		return err
	}
	_, err = sess.Finish(ctx, false)
	return err
}

func resolveMigrateFrom(proj *project.Project, from string) (project.Identity, error) {
	switch from {
	case "":
		switch proj.Identity {
		case project.IdentityNub, project.IdentityPNPM:
			return proj.Identity, nil
		default:
			return "", apperr.New(apperr.Usage, "lock.migrate", string(proj.Identity), "project identity is not nub or pnpm")
		}
	case "nub":
		return project.IdentityNub, nil
	case "pnpm":
		return project.IdentityPNPM, nil
	default:
		return "", apperr.New(apperr.Usage, "lock.migrate", from, "expected --from nub or pnpm")
	}
}

func lockIdentityFromBasename(name string) (project.Identity, bool) {
	switch name {
	case "m.lock":
		return project.IdentityMew, true
	case "nub.lock":
		return project.IdentityNub, true
	case "pnpm-lock.yaml":
		return project.IdentityPNPM, true
	default:
		return "", false
	}
}

// ValidateIncumbentLock parses the incumbent lock and optionally checks frozen drift.
func ValidateIncumbentLock(ctx context.Context, ac *Context, frozen bool) (string, error) {
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return "", err
	}
	path := LockPath(proj)
	ext, ok := lockfile.ExtAdapterFor(proj.Identity)
	if !ok {
		adapter := lockfile.AdapterFor(proj.Identity)
		if adapter == nil {
			return "", lockfile.NewUnsupported("lock.validate", path, "adapter not registered")
		}
		if _, err := adapter.Read(ctx, path); err != nil {
			return "", err
		}
	} else {
		if _, _, err := ext.ReadWithExtensions(ctx, path); err != nil {
			return "", err
		}
	}
	if frozen {
		if err := validateFrozenLockForProject(ctx, ac, proj); err != nil {
			return "", err
		}
	}
	return path, nil
}

// EncodeLossReportJSON returns a stable JSON loss report.
func EncodeLossReportJSON(report lockfile.LossReport) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "lock.migrate", "lossReport", err)
	}
	return buf.Bytes(), nil
}
