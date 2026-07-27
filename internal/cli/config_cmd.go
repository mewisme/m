package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/diagnostics"
	"github.com/mewisme/m/internal/project"
)

func newConfigCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get and set configuration",
	}
	cmd.AddCommand(newConfigGetCmd(g))
	cmd.AddCommand(newConfigSetCmd(g))
	cmd.AddCommand(newConfigListCmd(g))
	return cmd
}

func newConfigGetCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print an effective config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eff, err := loadEffective(g)
			if err != nil {
				return err
			}
			v, err := config.Get(eff, args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), formatConfigValue(v.Raw))
			return err
		},
	}
}

func newConfigSetCmd(g *globalFlags) *cobra.Command {
	var global bool
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config key in project or global JSONC",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, raw := args[0], args[1]
			val, err := config.ParseValue(key, raw)
			if err != nil {
				return apperr.Wrap(apperr.Usage, "config.set", key, err)
			}
			path, err := configWritePath(g, global)
			if err != nil {
				return err
			}
			return config.SetFile(path, key, val)
		},
	}
	cmd.Flags().BoolVar(&global, "global", false, "write user global config.jsonc")
	return cmd
}

func newConfigListCmd(g *globalFlags) *cobra.Command {
	var sources bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List effective configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			eff, err := loadEffective(g)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, e := range config.List(eff) {
				val := diagnostics.Redact(e.Value)
				if sources {
					fmt.Fprintf(out, "%s=%s\tsource=%s\tpath=%s\n", e.Key, val, e.Source, e.Path)
				} else {
					fmt.Fprintf(out, "%s=%s\n", e.Key, val)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&sources, "sources", false, "include source provenance")
	return cmd
}

// loadEffective rebuilds config for m config subcommands (not the mutation reload path).
func loadEffective(g *globalFlags) (*config.Effective, error) {
	cwd := g.cwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return nil, apperr.Wrap(apperr.IO, "config", "cwd", err)
		}
	}
	root := cwd
	if r, err := project.FindRoot(cwd); err == nil {
		root = r
	}

	cli := map[string]any{}
	if g.offline {
		cli["offline"] = true
	}
	if g.preferOffline {
		cli["prefer-offline"] = true
	}
	if g.configPath != "" {
		// m config resolves --config against CLI --cwd (not the mutation reload path).
		resolved, err := config.ResolveConfigPath(cwd, g.configPath)
		if err != nil {
			return nil, err
		}
		overlay, err := loadFileOverlay(resolved)
		if err != nil {
			return nil, err
		}
		for k, v := range overlay {
			cli[k] = v
		}
	}

	return config.Load(context.Background(), config.LoadOptions{
		CWD:         cwd,
		ProjectRoot: root,
		CLI:         cli,
		IdentityMew: true,
	})
}

func loadFileOverlay(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "config", path, err)
	}
	parsed, err := config.ParseJSONC(b)
	if err != nil {
		return nil, apperr.Wrap(apperr.Config, "config", path, err)
	}
	// Load via temp project file to reuse flatten/validate.
	dir, err := os.MkdirTemp("", "mew-cfg-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	tmp := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return nil, err
	}
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:         dir,
		ProjectRoot: dir,
		ProjectPath: tmp,
		GlobalPath:  filepath.Join(dir, "no-global.jsonc"),
		Env:         []string{},
	})
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for k, v := range eff.Values {
		if v.Source == config.SourceProject {
			out[k] = v.Raw
		}
	}
	_ = parsed
	return out, nil
}

func configWritePath(g *globalFlags, global bool) (string, error) {
	if global {
		return config.GlobalConfigPath(), nil
	}
	cwd := g.cwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	root, err := project.FindRoot(cwd)
	if err != nil {
		root = cwd
	}
	return filepath.Join(root, "m.jsonc"), nil
}

func formatConfigValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.Itoa(t)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	default:
		return fmt.Sprint(t)
	}
}
