package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/project"
)

// configScope is the target scope for config commands.
type configScope string

const (
	configScopeUser      configScope = "user"
	configScopeProject   configScope = "project"
	configScopeEffective configScope = "effective"
)

// configWriteTarget bundles scope and resolved file path.
type configWriteTarget struct {
	Scope configScope
	Path  string
}

// configWriteFlags holds the --scope flag shared by write commands.
type configWriteFlags struct {
	scope string
}

func (f *configWriteFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.scope, "scope", "user",
		"config scope: user or project (effective is read-only)")
}

func (f *configWriteFlags) resolvedScope() configScope {
	switch f.scope {
	case "user", "":
		return configScopeUser
	case "project":
		return configScopeProject
	default:
		return configScope(f.scope)
	}
}

func (f *configWriteFlags) validateWritable() error {
	switch f.resolvedScope() {
	case configScopeUser, configScopeProject:
		return nil
	case configScopeEffective:
		return apperr.New(apperr.Usage, "config.write", f.scope,
			"effective scope is read-only; use --scope user or --scope project")
	default:
		return apperr.New(apperr.Usage, "config.write", f.scope,
			"invalid scope: must be user, project, or effective")
	}
}

func (f *configWriteFlags) validateScope() error {
	switch f.resolvedScope() {
	case configScopeUser, configScopeProject, configScopeEffective:
		return nil
	default:
		return apperr.New(apperr.Usage, "config", f.scope,
			"invalid scope: must be user, project, or effective")
	}
}

// resolveConfigWriteTarget resolves the file path for a writable scope.
func resolveConfigWriteTarget(scope configScope, cwd string) (configWriteTarget, error) {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return configWriteTarget{}, apperr.Wrap(apperr.IO, "config.write", "cwd", err)
		}
	}

	switch scope {
	case configScopeUser:
		return configWriteTarget{
			Scope: configScopeUser,
			Path:  config.GlobalConfigPath(),
		}, nil
	case configScopeProject:
		root, err := project.FindRoot(cwd)
		if err != nil {
			return configWriteTarget{}, err
		}
		return configWriteTarget{
			Scope: configScopeProject,
			Path:  filepath.Join(root, "m.jsonc"),
		}, nil
	default:
		return configWriteTarget{}, apperr.New(apperr.Usage, "config.write", string(scope),
			"scope is not writable")
	}
}

// resolveEffectivePaths returns the ordered file paths that participate in effective resolution.
func resolveEffectivePaths(cwd string) (userPath, projectPath string) {
	userPath = config.GlobalConfigPath()
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if root, err := project.FindRoot(cwd); err == nil {
		projectPath = filepath.Join(root, "m.jsonc")
	}
	return
}

func displayConfigSource(src config.Source) string {
	if src == config.SourceGlobal {
		return "user"
	}
	return string(src)
}
