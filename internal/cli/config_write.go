package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/project"
)

type configWriteScope string

const (
	configWriteUser    configWriteScope = "user"
	configWriteProject configWriteScope = "project"
	configWriteFile    configWriteScope = "file"
)

type configWriteTarget struct {
	Scope configWriteScope
	Path  string
}

type configWriteOptions struct {
	Local, Global bool
	File, CWD     string
}

type configWriteFlags struct {
	local, global bool
	file          string
}

func (f *configWriteFlags) bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.local, "local", false, "write project-root m.jsonc")
	cmd.Flags().StringVar(&f.file, "file", "", "write exact config file path")
	cmd.Flags().BoolVar(&f.global, "global", false, "write user config (deprecated; default)")
	_ = cmd.Flags().MarkDeprecated("global", "user scope is the default; omit --global")
}

func (f *configWriteFlags) options(g *globalFlags) configWriteOptions {
	return configWriteOptions{
		Local:  f.local,
		Global: f.global,
		File:   f.file,
		CWD:    g.cwd,
	}
}

// resolveConfigWriteTarget picks the user, project, or explicit file write path.
func resolveConfigWriteTarget(opts configWriteOptions) (configWriteTarget, error) {
	hasLocal := opts.Local
	hasFile := opts.File != ""
	hasGlobal := opts.Global

	if hasGlobal && (hasLocal || hasFile) {
		return configWriteTarget{}, apperr.New(apperr.Usage, "config.write", "",
			"--global cannot be combined with --local or --file")
	}
	if hasLocal && hasFile {
		return configWriteTarget{}, apperr.New(apperr.Usage, "config.write", "",
			"--local and --file are mutually exclusive")
	}

	cwd := opts.CWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return configWriteTarget{}, apperr.Wrap(apperr.IO, "config.write", "cwd", err)
		}
	}

	if hasFile {
		path, err := config.ResolveConfigPath(cwd, opts.File)
		if err != nil {
			return configWriteTarget{}, apperr.Wrap(apperr.Usage, "config.write", opts.File, err)
		}
		if st, err := os.Stat(path); err == nil && st.IsDir() {
			return configWriteTarget{}, apperr.New(apperr.IO, "config.write", path, "path is a directory")
		}
		return configWriteTarget{Scope: configWriteFile, Path: path}, nil
	}

	if hasLocal {
		root, err := project.FindRoot(cwd)
		if err != nil {
			return configWriteTarget{}, err
		}
		return configWriteTarget{
			Scope: configWriteProject,
			Path:  filepath.Join(root, "m.jsonc"),
		}, nil
	}

	// Default and deprecated --global: ambient user config path.
	return configWriteTarget{
		Scope: configWriteUser,
		Path:  config.GlobalConfigPath(),
	}, nil
}

func displayConfigSource(src config.Source) string {
	if src == config.SourceGlobal {
		return "user"
	}
	return string(src)
}
