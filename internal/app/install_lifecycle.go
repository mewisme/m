package app

import (
	"context"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lifecycle"
	"github.com/mewisme/mew/internal/linker"
	"github.com/mewisme/mew/internal/process"
	"github.com/mewisme/mew/internal/project"
)

func runLifecyclePhase(ctx context.Context, ac *Context, proj *project.Project, opts InstallOptions, stageNM string, g *graph.Graph, linkPlan *linker.Plan) error {
	if opts.IgnoreScripts || config.Bool(ac.Config, "lifecycle.ignore_scripts", false) {
		return nil
	}
	if !lifecycle.Enabled(ac.Config) {
		return nil
	}
	emitPhase(ac, "lifecycle", "")
	trust, err := lifecycle.LoadTrust(proj.Root)
	if err != nil {
		return err
	}
	src := process.EnvSource{Explicit: true}
	if ac.Config != nil && ac.Config.Env.Initialized() {
		src.Vars = ac.Config.Env.Environ()
	}
	_, err = lifecycle.RunInstallScripts(ctx, lifecycle.InstallInput{
		ProjectRoot: proj.Root,
		NodeModules: stageNM,
		Graph:       g,
		LinkPlan:    linkPlan,
		Config:      ac.Config,
		Env:         src,
		Trusted:     trust,
		AuditPath:   lifecycle.AuditFilePath(proj.Root),
		CacheDir:    lifecycle.CacheDir(ac.Config),
	})
	return err
}
