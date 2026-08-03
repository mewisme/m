package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/transaction"
)

// MigrateLockOptions controls explicit lock migration to m.lock.
type MigrateLockOptions struct {
	From      string // nub|pnpm|npm|bun|yarn; empty auto-detects source
	DryRun    bool
	PnpmMajor int
}

// MigrateLockResult summarizes migrate output.
type MigrateLockResult struct {
	DryRun           bool
	LossReport       lockfile.LossReport
	Path             string
	SourceIdentity   string
	SourceLockPath   string
	Detection        lockfile.Detection
	PreservedUnknown int
}

// LockDiffOptions compares lock graphs.
type LockDiffOptions struct {
	OtherPath string
	FromPath  string
	ToPath    string
	PnpmMajor int
}

// LockDiff compares lock graphs.
func LockDiff(ctx context.Context, ac *Context, opts LockDiffOptions) (*lockfile.GraphDiff, error) {
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return nil, err
	}
	hints := lockHintsFromProject(proj)
	if opts.FromPath != "" || opts.ToPath != "" {
		if opts.FromPath == "" || opts.ToPath == "" {
			return nil, apperr.New(apperr.Usage, "lock.diff", "", "both --from and --to are required")
		}
		leftPath, err := resolveLockDiffPath(ac, opts.FromPath)
		if err != nil {
			return nil, err
		}
		rightPath, err := resolveLockDiffPath(ac, opts.ToPath)
		if err != nil {
			return nil, err
		}
		left, err := readLockGraphAtPath(ctx, leftPath, hints, opts.PnpmMajor)
		if err != nil {
			return nil, err
		}
		right, err := readLockGraphAtPath(ctx, rightPath, hints, opts.PnpmMajor)
		if err != nil {
			return nil, err
		}
		return lockfile.DiffGraphs(left, right)
	}
	if proj.Identity == project.IdentityPNPM {
		prior, readErr := project.ReadLockfileBytes(proj.Root, proj.Identity)
		if readErr != nil {
			return nil, readErr
		}
		if _, err := detectPnpmLock(prior, proj, opts.PnpmMajor); err != nil {
			return nil, err
		}
	}
	left, err := lockfile.ReadGraph(ctx, proj.Root, proj.Identity)
	if err != nil {
		return nil, err
	}
	otherPath := opts.OtherPath
	if otherPath == "" {
		otherPath = filepath.Join(proj.Root, "m.lock")
	} else if !filepath.IsAbs(otherPath) {
		otherPath = filepath.Join(ac.CWD, otherPath)
	}
	right, err := readLockGraphAtPath(ctx, otherPath, hints, opts.PnpmMajor)
	if err != nil {
		return nil, err
	}
	return lockfile.DiffGraphs(left, right)
}

func resolveLockDiffPath(ac *Context, path string) (string, error) {
	if path == "" {
		return "", apperr.New(apperr.Usage, "lock.diff", "", "empty lock path")
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Join(ac.CWD, path), nil
}

func readLockGraphAtPath(ctx context.Context, path string, hints lockfile.ProjectHints, pnpmMajor int) (*graph.Graph, error) {
	id, ok := lockIdentityFromBasename(filepath.Base(path))
	if !ok {
		return nil, apperr.New(apperr.Usage, "lock.diff", path, "unrecognized lockfile name")
	}
	if id == project.IdentityPNPM {
		prior, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, apperr.Wrap(apperr.IO, "lock.diff", path, readErr)
		}
		if _, err := detectPnpmLockBytes(prior, hints, pnpmMajor); err != nil {
			return nil, err
		}
	}
	adapter := lockfile.AdapterFor(id)
	if adapter == nil {
		return nil, lockfile.NewUnsupported("lock.diff", path, "adapter not registered")
	}
	return adapter.Read(ctx, path)
}

// MigrateLock migrates an incumbent nub/pnpm lock to m.lock.
func MigrateLock(ctx context.Context, ac *Context, opts MigrateLockOptions) (MigrateLockResult, error) {
	var out MigrateLockResult
	proj, err := OpenProjectForMigrate(ctx, ac)
	if err != nil {
		return out, err
	}
	fromID, lockPath, err := project.ResolveMigrateSource(proj.Root, opts.From)
	if err != nil {
		return out, err
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
	migrateExt, err := preservePnpmMigrateLoss(fromID, prior, &loss)
	if err != nil {
		return out, err
	}
	mext, ok := lockfile.ExtAdapterFor(project.IdentityMew)
	if !ok {
		return out, lockfile.NewUnsupported("lock.migrate", "m.lock", "m.lock adapter not registered")
	}
	det := lockfile.Detection{Format: "nub", ProducerMajor: 9, Confidence: lockfile.DetectionCertain}
	if fromID == project.IdentityPNPM {
		det, err = detectPnpmLock(prior, proj, opts.PnpmMajor)
		if err != nil {
			return out, err
		}
	}
	if fromID == project.IdentityNPM {
		det, err = detectNpmLock(prior)
		if err != nil {
			return out, err
		}
	}
	if fromID == project.IdentityBun {
		det, err = detectBunLock(prior)
		if err != nil {
			return out, err
		}
	}
	if fromID == project.IdentityYarn {
		det, err = detectYarnLock(prior, proj.Root)
		if err != nil {
			return out, err
		}
	}
	encodeRes, encErr := lockfile.EncodePreserving(ctx, mext, filepath.Join(proj.Root, "m.lock"), g, nil, migrateExt, det)
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
	out.SourceIdentity = string(fromID)
	out.SourceLockPath = lockPath
	out.Detection = det
	for _, item := range loss.Items {
		if item.Category == "extension" ||
			item.Reason == "top-level extension not mapped to canonical graph" ||
			item.Reason == "nub extension not mapped to canonical graph" {
			out.PreservedUnknown++
		}
	}
	if opts.DryRun {
		out.DryRun = true
		return out, nil
	}
	if encErr != nil {
		return out, encErr
	}
	if semantic := lockfileSemanticLoss(loss); len(semantic) > 0 {
		loss.Items = semantic
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
	foreign := foreignLocksPresent(proj.Root)
	for _, name := range foreign {
		plan = append(plan, transaction.Op{Kind: transaction.OpRemove, Path: name})
	}
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
	for _, name := range foreign {
		if err := txn.RecordBackup(name); err != nil {
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

// foreignLocksPresent lists non-m.lock identity lockfiles that must leave after migrate.
func foreignLocksPresent(root string) []string {
	names := []string{
		"nub.lock", "pnpm-lock.yaml", "package-lock.json", "npm-shrinkwrap.json",
		"yarn.lock", "bun.lock", "bun.lockb",
	}
	var out []string
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			out = append(out, name)
		}
	}
	return out
}

func lockfileSemanticLoss(loss lockfile.LossReport) []lockfile.LossItem {
	out := make([]lockfile.LossItem, 0, len(loss.Items))
	for _, item := range loss.Items {
		if item.Semantic {
			out = append(out, item)
		}
	}
	return out
}

func lockIdentityFromBasename(name string) (project.Identity, bool) {
	switch name {
	case "m.lock":
		return project.IdentityMew, true
	case "nub.lock":
		return project.IdentityNub, true
	case "pnpm-lock.yaml":
		return project.IdentityPNPM, true
	case "package-lock.json", "npm-shrinkwrap.json":
		return project.IdentityNPM, true
	case "bun.lock":
		return project.IdentityBun, true
	case "yarn.lock":
		return project.IdentityYarn, true
	}
	type suffixID struct {
		suffix string
		id     project.Identity
	}
	for _, entry := range []suffixID{
		{suffix: "m.lock", id: project.IdentityMew},
		{suffix: "nub.lock", id: project.IdentityNub},
		{suffix: "pnpm-lock.yaml", id: project.IdentityPNPM},
		{suffix: "package-lock.json", id: project.IdentityNPM},
		{suffix: "npm-shrinkwrap.json", id: project.IdentityNPM},
		{suffix: "bun.lock", id: project.IdentityBun},
		{suffix: "yarn.lock", id: project.IdentityYarn},
	} {
		if strings.HasSuffix(name, entry.suffix) {
			return entry.id, true
		}
	}
	return "", false
}

// ValidateLockOptions controls incumbent lock validation.
type ValidateLockOptions struct {
	Frozen    bool
	PnpmMajor int
}

// ValidateLockResult reports incumbent lock validation outcome.
type ValidateLockResult struct {
	Path      string
	Detection lockfile.Detection
}

// ValidateIncumbentLock parses the incumbent lock and optionally checks frozen drift.
func ValidateIncumbentLock(ctx context.Context, ac *Context, opts ValidateLockOptions) (ValidateLockResult, error) {
	var out ValidateLockResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return out, err
	}
	path := LockPath(proj)
	out.Path = path
	ext, ok := lockfile.ExtAdapterFor(proj.Identity)
	if !ok {
		adapter := lockfile.AdapterFor(proj.Identity)
		if adapter == nil {
			return out, lockfile.NewUnsupported("lock.validate", path, "adapter not registered")
		}
		if _, err := adapter.Read(ctx, path); err != nil {
			return out, err
		}
	} else {
		if _, _, err := ext.ReadWithExtensions(ctx, path); err != nil {
			return out, err
		}
	}
	if det, derr := detectIncumbentLock(proj); derr == nil {
		out.Detection = det
	}
	if proj.Identity == project.IdentityPNPM {
		prior, readErr := project.ReadLockfileBytes(proj.Root, proj.Identity)
		if readErr != nil {
			return out, readErr
		}
		det, derr := detectPnpmLock(prior, proj, opts.PnpmMajor)
		if derr != nil {
			return out, derr
		}
		out.Detection = det
	}
	if proj.Identity == project.IdentityNPM {
		prior, readErr := project.ReadLockfileBytes(proj.Root, proj.Identity)
		if readErr != nil {
			return out, readErr
		}
		det, derr := detectNpmLock(prior)
		if derr != nil {
			return out, derr
		}
		out.Detection = det
	}
	if opts.Frozen {
		if err := validateFrozenLockForProject(ctx, ac, proj); err != nil {
			return out, err
		}
	}
	return out, nil
}

func detectIncumbentLock(proj *project.Project) (lockfile.Detection, error) {
	prior, err := project.ReadLockfileBytes(proj.Root, proj.Identity)
	if err != nil {
		return lockfile.Detection{}, err
	}
	switch proj.Identity {
	case project.IdentityPNPM:
		return detectPnpmLock(prior, proj, 0)
	case project.IdentityNPM:
		return detectNpmLock(prior)
	case project.IdentityBun:
		return detectBunLock(prior)
	case project.IdentityYarn:
		return detectYarnLock(prior, proj.Root)
	case project.IdentityNub:
		det, err := detectPnpmLock(prior, proj, 0)
		if err != nil {
			return det, err
		}
		det.Format = "nub"
		return det, nil
	default:
		return lockfile.Detection{Format: "mew", Confidence: lockfile.DetectionCertain}, nil
	}
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
