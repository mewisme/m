package envexec

import (
	"context"
	"os"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/process"
	"github.com/mewisme/mew/internal/runner"
)

// Orchestrator coordinates validation, planning, materialization, leasing, and execution.
type Orchestrator struct {
	Providers ProviderRegistry
	Executor  ExecAdapter
	Reporter  diagnostics.Reporter
	Leases    LeaseManager
}

// Execute runs the unified execution flow:
// validate → select provider → Plan → Materialize → verify → lease → runner.Exec.
func (o *Orchestrator) Execute(
	ctx context.Context,
	deps ProviderDeps,
	req ExecutionRequest,
) (runner.ExecResult, error) {
	var empty runner.ExecResult
	if err := ValidateRequest(req); err != nil {
		return empty, err
	}
	if err := ValidatePolicy(req.Policy); err != nil {
		return empty, err
	}

	provider, err := o.Providers.providerFor(req)
	if err != nil {
		return empty, err
	}

	plan, err := provider.Plan(ctx, deps, req)
	if err != nil {
		return empty, err
	}

	if plan.DLXResolved != nil {
		if plan.DLXResolved.Command != "" {
			req.Command.Name = plan.DLXResolved.Command
		}
		if req.Command.OwnerDependency == "" {
			req.Command.OwnerDependency = plan.DLXResolved.Owner
		}
	}

	var env PreparedEnvironment
	if plan.Prepared != nil {
		env = *plan.Prepared
	} else {
		env, err = provider.Materialize(ctx, deps, plan)
		if err != nil {
			return empty, err
		}
	}
	if env.CommandOwner != "" && req.Command.OwnerDependency == "" {
		req.Command.OwnerDependency = env.CommandOwner
	}
	if req.Command.Name == "" && env.InferredCommand != "" {
		req.Command.Name = env.InferredCommand
	}

	if err := verifyPrepared(env); err != nil {
		o.runCleanup(env)
		return empty, err
	}

	releaseLease, err := o.acquireLease(ctx, env)
	if err != nil {
		o.runCleanup(env)
		return empty, err
	}
	if releaseLease != nil {
		defer releaseLease()
	}

	if err := o.emitPrepared(req, env); err != nil {
		o.runCleanup(env)
		return empty, err
	}

	exec := o.Executor
	if exec == nil {
		exec = DefaultExecAdapter{}
	}
	sup := process.NewExecSupervisor()
	result, execErr := exec.Exec(ctx, env, req, sup)
	o.runCleanup(env)
	return result, execErr
}

func verifyPrepared(env PreparedEnvironment) error {
	if env.Root == "" || env.NodeModules == "" {
		return apperr.New(apperr.Internal, "envexec.verify", "", "incomplete prepared environment")
	}
	return nil
}

func (o *Orchestrator) emitPrepared(req ExecutionRequest, env PreparedEnvironment) error {
	if o.Reporter == nil {
		return nil
	}
	ev, err := buildEnvironmentPreparedEvent(req, env, 0)
	if err != nil {
		return err
	}
	return o.Reporter.EnvironmentPrepared(ev)
}

func (o *Orchestrator) acquireLease(ctx context.Context, env PreparedEnvironment) (func(), error) {
	if env.LeasePolicy == LeaseNone || o.Leases == nil {
		return nil, nil
	}
	pid := os.Getpid()
	token := time.Now().UnixNano()
	release, err := o.Leases.Acquire(ctx, env.Identity, "envexec", pid, token)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "envexec.lease", env.Identity.IdentityDigest(), err)
	}
	return release, nil
}

func (o *Orchestrator) runCleanup(env PreparedEnvironment) {
	if env.Cleanup == nil {
		return
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = env.Cleanup(cleanupCtx)
}
