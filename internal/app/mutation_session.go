package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/transaction"
)

// MutationSession holds project mutation ownership from BeginMutation until Finish.
type MutationSession struct {
	ac          *Context
	projectRoot string
	runner      *transaction.Runner
	proj        *project.Project
	effective   *config.Effective
	sessionAC   *Context
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

// ReloadEffectiveConfig reloads project config after mutation ownership is held.
func (s *MutationSession) ReloadEffectiveConfig(ctx context.Context) error {
	if s == nil {
		return apperr.New(apperr.Internal, "app.mutation", "", "nil session")
	}
	if s.ac == nil || s.ac.Config == nil {
		return apperr.New(apperr.Internal, "app.mutation", "", "missing app context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	eff, err := config.Load(ctx, config.LoadOptions{
		CWD:         s.ac.CWD,
		ProjectRoot: s.projectRoot,
		Env:         os.Environ(),
		CLI:         cliOverlayFromEffective(s.ac.Config),
	})
	if err != nil {
		return err
	}
	s.effective = eff
	s.sessionAC = nil
	return nil
}

// AppContext returns a shallow copy of the session app context with reloaded config.
// Never mutates the shared context passed to BeginMutationSession.
// Call ReopenProject (or ReloadEffectiveConfig) before AppContext.
func (s *MutationSession) AppContext() (*Context, error) {
	if s == nil || s.ac == nil {
		return nil, apperr.New(apperr.Internal, "app.mutation", "", "nil session")
	}
	if s.effective == nil {
		return nil, apperr.New(apperr.Internal, "app.mutation", "", "effective config not loaded; call ReopenProject first")
	}
	if s.sessionAC != nil && s.sessionAC.Config == s.effective {
		return s.sessionAC, nil
	}
	s.sessionAC = &Context{
		CWD:       s.ac.CWD,
		Config:    s.effective,
		Reporter:  s.ac.Reporter,
		Version:   s.ac.Version,
		Commit:    s.ac.Commit,
		BuildDate: s.ac.BuildDate,
		Ctx:       s.ac.Ctx,
	}
	return s.sessionAC, nil
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
	if err := s.ReloadEffectiveConfig(ctx); err != nil {
		return nil, err
	}
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
// Repeated calls are no-ops once the runner has been cleared.
func (s *MutationSession) Abort(ctx context.Context) (transaction.FinishResult, error) {
	if s == nil || s.runner == nil {
		return transaction.FinishResult{}, nil
	}
	fr, cleanupErr, _ := rollbackSession(ctx, s, s.runner)
	return fr, cleanupErr
}

func cliOverlayFromEffective(eff *config.Effective) map[string]any {
	if eff == nil || eff.Values == nil {
		return nil
	}
	out := make(map[string]any)
	for k, v := range eff.Values {
		if v.Source == config.SourceCLI {
			out[k] = v.Raw
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
