package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/presentation"
	"github.com/mewisme/mew/internal/project"
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
			r := g.mustStaticRenderer(cmd, eff)
			entries := config.List(eff)
			cols := []presentation.TableColumn{
				{Key: "key", Header: "KEY", MinWidth: 8, Prefer: 28, Primary: true, Truncate: presentation.TruncateMiddle},
				{Key: "value", Header: "VALUE", MinWidth: 4, Prefer: 32, Truncate: presentation.TruncateMiddle},
				{Key: "env", Header: "ENV", MinWidth: 3, Prefer: 28, Truncate: presentation.TruncateMiddle},
			}
			if sources {
				cols = append(cols,
					presentation.TableColumn{Key: "source", Header: "SOURCE", MinWidth: 4, Prefer: 12},
					presentation.TableColumn{Key: "path", Header: "PATH", MinWidth: 4, Prefer: 24, Truncate: presentation.TruncateMiddle},
				)
			}
			rows := make([]map[string]string, 0, len(entries))
			for _, e := range entries {
				env := e.Env
				if env == "" {
					env = "-"
				}
				row := map[string]string{
					"key":   e.Key,
					"value": diagnostics.Redact(e.Value),
					"env":   env,
				}
				if sources {
					row["source"] = string(e.Source)
					row["path"] = e.Path
				}
				rows = append(rows, row)
			}
			return writeStaticOut(cmd, r.Table(presentation.TableModel{Columns: cols, Rows: rows}))
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
		// intentional: m config set --global writes ambient user config path.
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
