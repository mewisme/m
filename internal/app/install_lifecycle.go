package app

import (
	"context"
	"fmt"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lifecycle"
	"github.com/mewisme/mew/internal/linker"
	"github.com/mewisme/mew/internal/process"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/prompt"
)

func runLifecyclePhase(ctx context.Context, ac *Context, proj *project.Project, opts InstallOptions, stageNM string, g *graph.Graph, linkPlan *linker.Plan, opID string, res *InstallResult) error {
	if opts.IgnoreScripts || config.Bool(ac.Config, "lifecycle.ignore_scripts", false) {
		ph := beginInstallPhase(ac, opID, phaseLifecycle)
		ph.Complete(statusSkipped)
		return nil
	}
	if !lifecycle.Enabled(ac.Config) {
		ph := beginInstallPhase(ac, opID, phaseLifecycle)
		ph.Complete(statusSkipped)
		return nil
	}
	ph := beginInstallPhase(ac, opID, phaseLifecycle)
	var lifeErr error
	defer func() { ph.CompleteErr(lifeErr) }()
	trust, err := lifecycle.LoadTrust(proj.Root)
	if err != nil {
		lifeErr = err
		return err
	}
	src := process.EnvSource{Explicit: true}
	if ac.Config != nil && ac.Config.Env.Initialized() {
		src.Vars = ac.Config.Env.Environ()
	}
	if ac.SuspendUI != nil {
		_ = ac.SuspendUI(ctx)
	}
	if ac.ResumeUI != nil {
		defer func() { _ = ac.ResumeUI(ctx) }()
	}
	lifeRes, err := lifecycle.RunInstallScripts(ctx, lifecycle.InstallInput{
		ProjectRoot: proj.Root,
		NodeModules: stageNM,
		Graph:       g,
		LinkPlan:    linkPlan,
		Config:      ac.Config,
		Env:         src,
		Trusted:     trust,
		Interactive: ac != nil && ac.CanPrompt,
		Prompter:    acPrompter(ac),
		AllowOnce:   map[string]struct{}{},
		AuditPath:   lifecycle.AuditFilePath(proj.Root),
		CacheDir:    lifecycle.CacheDir(ac.Config),
	})
	if res != nil {
		res.ScriptsRun += lifeRes.Ran
		res.ScriptsBlocked += lifeRes.Skipped
	}
	if lifeRes.Skipped > 0 {
		bin := ac.BinaryName
		if bin == "" {
			bin = "m"
		}
		emitInstallNotice(ac, "warning", "ERR_M_LIFECYCLE",
			fmt.Sprintf("%d lifecycle scripts were blocked", lifeRes.Skipped),
			"Run `"+bin+" builds` to review them")
	}
	if err != nil {
		lifeErr = err
		emitInstallNotice(ac, "security-warning", "ERR_M_LIFECYCLE",
			"lifecycle script failed", "")
		return err
	}
	lifeErr = nil
	ph.Complete(statusOK,
		diagnostics.Metric{Name: "ran", Value: float64(lifeRes.Ran), Unit: "scripts"},
		diagnostics.Metric{Name: "blocked", Value: float64(lifeRes.Skipped), Unit: "scripts"},
	)
	return nil
}

func acPrompter(ac *Context) prompt.Prompter {
	if ac == nil {
		return nil
	}
	return ac.Prompter
}
