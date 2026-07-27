package lifecycle

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/process"
)

// RunInstallScripts executes discovered lifecycle scripts under policy.
func RunInstallScripts(ctx context.Context, in InstallInput) (Result, error) {
	var res Result
	if in.Supervisor == nil {
		in.Supervisor = process.NewExecSupervisor()
	}
	plan, err := Discover(in.Graph, in.LinkPlan)
	if err != nil {
		return res, err
	}
	seen := map[string]struct{}{}
	for _, script := range plan.Scripts {
		trustKey := script.PackageName
		if _, ok := seen[trustKey]; !ok {
			seen[trustKey] = struct{}{}
			if err := CheckTrust(trustKey, in.Config, in.Trusted, in.Interactive, nil, nil); err != nil {
				return res, err
			}
		}
		if script.Name == "prepare" {
			if hit, err := cacheHit(in.CacheDir, script); err != nil {
				return res, err
			} else if hit {
				res.Cached++
				continue
			}
		}
		start := time.Now()
		exitCode, runErr := RunScript(ctx, in.Supervisor, RunSpec{
			Script:      script,
			NodeModules: in.NodeModules,
			Env:         in.Env,
		})
		dur := time.Since(start)
		if in.AuditPath != "" {
			if err := AppendAudit(in.AuditPath, AuditEntry{
				Package:    script.PackageName,
				Script:     script.Name,
				ExitCode:   exitCode,
				DurationMs: dur.Milliseconds(),
			}); err != nil {
				return res, err
			}
		}
		if runErr != nil {
			return res, runErr
		}
		if script.Name == "prepare" {
			if err := markCache(in.CacheDir, script); err != nil {
				return res, err
			}
		}
		res.Ran++
	}
	return res, nil
}

// RunSpec describes one script execution.
type RunSpec struct {
	Script      Script
	NodeModules string
	Env         []string
}

// RunScript executes one lifecycle script in a restricted environment.
func RunScript(ctx context.Context, sup process.ProcessSupervisor, spec RunSpec) (int, error) {
	binDir := process.BinDirForPackage(spec.Script.PackageDir, spec.NodeModules)
	env := process.RestrictedEnv(spec.Env, binDir)
	execSpec := process.Spec{
		Path: spec.Script.Command,
		Dir:  spec.Script.PackageDir,
		Env:  env,
	}
	h, err := sup.Start(ctx, execSpec)
	if err != nil {
		return 1, apperr.Wrap(apperr.Install, "lifecycle.run", spec.Script.PackageName, err)
	}
	waitErr := sup.Wait(ctx, h)
	code := process.ExitCode(waitErr)
	if waitErr != nil && code == 0 {
		code = 1
	}
	if code != 0 {
		return code, apperr.New(apperr.Install, "lifecycle.run", spec.Script.PackageName,
			fmt.Sprintf("%s script %s failed with exit %d", spec.Script.PackageName, spec.Script.Name, code))
	}
	return 0, nil
}

// AuditFilePath returns <project>/.mew/lifecycle-audit.jsonl.
func AuditFilePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".mew", "lifecycle-audit.jsonl")
}

// CacheDir returns the lifecycle build cache directory under cache root.
func CacheDir(eff *config.Effective) string {
	return filepath.Join(config.CacheRoot(eff), "lifecycle")
}
