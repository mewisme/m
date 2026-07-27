package app

import (
	"context"

	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/linker"
	"github.com/mewisme/m/internal/linker/hoisted"
	"github.com/mewisme/m/internal/linker/isolated"
	"github.com/mewisme/m/internal/linker/planner"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/workspace"
)

type linkerOpts struct {
	NodeModules  string
	ExtractDirs  map[string]string
	Capabilities planner.Capabilities
	UseSmartLink bool
}

func newLinker(mode string, opts linkerOpts) linker.Linker {
	switch mode {
	case "isolated":
		return &isolated.Linker{
			NodeModules:  opts.NodeModules,
			ExtractDirs:  opts.ExtractDirs,
			Capabilities: opts.Capabilities,
			UseSmartLink: opts.UseSmartLink,
		}
	default:
		return &hoisted.Linker{
			NodeModules:  opts.NodeModules,
			ExtractDirs:  opts.ExtractDirs,
			Capabilities: opts.Capabilities,
			UseSmartLink: opts.UseSmartLink,
		}
	}
}

func resolveLinkerMode(ctx context.Context, ac *Context, proj *project.Project, opts InstallOptions) (string, error) {
	_ = ctx
	if ac != nil && ac.Config != nil && workspace.Enabled(ac.Config) && opts.Linker == "" {
		if patterns, err := proj.Doc.WorkspacePatterns(); err == nil && len(patterns) > 0 {
			if config.String(ac.Config, "install.linker", "auto") == "auto" {
				ac.Config.Values["install.linker"] = config.Value{Raw: "isolated", Source: config.SourceCLI, Path: "workspaces"}
			}
		}
	}
	lockLinker := ""
	if data, err := readLockSettings(proj.Root); err == nil && data != nil {
		lockLinker = data.Settings.Linker
	}
	if opts.Linker != "" && ac != nil && ac.Config != nil {
		ac.Config.Values["install.linker"] = config.Value{Raw: opts.Linker, Source: config.SourceCLI, Path: "cli"}
	}
	return config.ResolveLinkerMode(ac.Config, lockLinker, opts.Frozen)
}
