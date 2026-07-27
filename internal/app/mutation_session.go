package app

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/transaction"
)

// MutationSession holds project mutation ownership from BeginMutation until Finish.
type MutationSession struct {
	ac          *Context
	projectRoot string
	runner      *transaction.Runner
	proj        *project.Project
}

// BeginMutationSession acquires the project lock, recovers incomplete transactions, and
// starts a new mutation journal. No live manifest or lock reads occur before this returns.
func BeginMutationSession(ctx context.Context, ac *Context, projectRoot string) (*MutationSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, apperr.Wrap(apperr.Cancelled, "app.mutation", "", err)
	}
	if ac == nil || ac.Config == nil {
		return nil, apperr.New(apperr.Internal, "app.mutation", "", "missing app context")
	}
	root, err := resolveProjectRoot(ac, projectRoot)
	if err != nil {
		return nil, err
	}
	run, err := transaction.BeginMutation(ctx, root)
	if err != nil {
		return nil, err
	}
	return &MutationSession{ac: ac, projectRoot: root, runner: run}, nil
}

// Runner returns the active transaction runner (lock held until Finish).
func (s *MutationSession) Runner() *transaction.Runner {
	if s == nil {
		return nil
	}
	return s.runner
}

// ReopenProject reads live package.json, lock hints, and config after ownership is held.
func (s *MutationSession) ReopenProject(ctx context.Context) (*project.Project, error) {
	if s == nil {
		return nil, apperr.New(apperr.Internal, "app.mutation", "", "nil session")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	proj, err := project.Open(ctx, s.projectRoot)
	if err != nil {
		return nil, err
	}
	s.proj = proj
	return proj, nil
}

// Project returns the project loaded by the most recent ReopenProject call.
func (s *MutationSession) Project() *project.Project {
	if s == nil {
		return nil
	}
	return s.proj
}

// Finish completes the transaction journal and releases the session-owned project lock.
func (s *MutationSession) Finish(ctx context.Context, keepJournal bool) (transaction.FinishResult, error) {
	if s == nil || s.runner == nil {
		return transaction.FinishResult{}, nil
	}
	if err := ctx.Err(); err != nil {
		return transaction.FinishResult{}, err
	}
	txnID := s.runner.ID
	fr := s.runner.Finish(keepJournal, transaction.DefaultFinishOpts())
	if err := releaseSessionLock(s.projectRoot, txnID, &fr); err != nil {
		if fr.HasCriticalCleanupFailure() {
			return fr, err
		}
	}
	s.runner = nil
	if fr.HasCriticalCleanupFailure() {
		return fr, apperr.New(apperr.Transaction, "app.mutation.finish", "", "transaction cleanup incomplete")
	}
	return fr, nil
}

// Abort rolls back an in-progress transaction and releases the session-owned project lock.
func (s *MutationSession) Abort(ctx context.Context) (transaction.FinishResult, error) {
	if s == nil || s.runner == nil {
		return transaction.FinishResult{}, nil
	}
	txnID := s.runner.ID
	fr, err := s.runner.Rollback(ctx, transaction.DefaultFinishOpts())
	lockErr := releaseSessionLock(s.projectRoot, txnID, &fr)
	s.runner = nil
	return fr, errors.Join(err, lockErr)
}

func resolveProjectRoot(ac *Context, explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	if ac == nil || ac.CWD == "" {
		return "", apperr.New(apperr.Internal, "app.mutation", "", "missing project cwd")
	}
	return project.FindRoot(ac.CWD)
}

// usesStagedSnapshotInputs is true when install should trust staged snapshot bytes
// instead of validating the live lockfile (snapshot restore; full pair validation is phase 4).
func usesStagedSnapshotInputs(opts InstallOptions) bool {
	return opts.PreResolvedGraph != nil &&
		(len(opts.StagedManifest) > 0 || len(opts.StagedLock) > 0)
}

func isLockWaitCancellation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return apperr.CodeOf(err) == apperr.Cancelled
}
