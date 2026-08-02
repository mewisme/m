package cli

import (
	"context"
	"encoding/json"
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

// userScopedKeys are config keys that can only be written to user config,
// never project config. Attempting --local for these keys is rejected.
var userScopedKeys = map[string]bool{
	"ui.theme": true,
}

func newConfigCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get and set configuration",
	}
	cmd.AddCommand(newConfigGetCmd(g))
	cmd.AddCommand(newConfigSetCmd(g))
	cmd.AddCommand(newConfigUnsetCmd(g))
	cmd.AddCommand(newConfigListCmd(g))
	cmd.AddCommand(newConfigPathCmd(g))
	cmd.AddCommand(newConfigPathsCmd(g))
	return cmd
}

func newConfigGetCmd(g *globalFlags) *cobra.Command {
	var showSource bool
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Print an effective config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eff, err := loadEffective(g)
			if err != nil {
				return err
			}
			key := args[0]
			v, err := config.Get(eff, key)
			if err != nil {
				return err
			}
			val := formatConfigValue(v.Raw)
			if !showSource {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), val)
				return err
			}
			src := displayConfigSource(v.Source)
			if configOutputStructured(g, cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(map[string]any{
					"key":    key,
					"value":  v.Raw,
					"source": src,
					"path":   v.Path,
				})
			}
			r := g.mustStaticRenderer(cmd)
			out := val + "\n\n" + r.KeyValues([]presentation.KeyValue{
				{Key: "Source", Value: src},
				{Key: "Path", Value: v.Path, Style: presentation.ValuePath},
			})
			return writeStaticOut(cmd, out)
		},
	}
	cmd.Flags().BoolVar(&showSource, "source", false, "include provenance source and path")
	return cmd
}

func newConfigSetCmd(g *globalFlags) *cobra.Command {
	var flags configWriteFlags
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config key (default: user config)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, raw := args[0], args[1]
			val, err := config.ParseValue(key, raw)
			if err != nil {
				return apperr.Wrap(apperr.Usage, "config.set", key, err)
			}
			target, err := resolveConfigWriteTarget(flags.options(g))
			if err != nil {
				return err
			}
			if err := checkUserScopedKey(key, target); err != nil {
				return err
			}
			if err := config.SetFile(target.Path, key, val); err != nil {
				return err
			}
			return writeConfigMutationResult(g, cmd, "Set", key, formatConfigValue(val), target)
		},
	}
	flags.bind(cmd)
	return cmd
}

func newConfigUnsetCmd(g *globalFlags) *cobra.Command {
	var flags configWriteFlags
	cmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Remove a config key (default: user config)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			target, err := resolveConfigWriteTarget(flags.options(g))
			if err != nil {
				return err
			}
			if err := checkUserScopedKey(key, target); err != nil {
				return err
			}
			if err := config.UnsetFile(target.Path, key); err != nil {
				return err
			}
			return writeConfigMutationResult(g, cmd, "Removed", key, "", target)
		},
	}
	flags.bind(cmd)
	return cmd
}

func newConfigPathCmd(g *globalFlags) *cobra.Command {
	var flags configWriteFlags
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Print a config write-target path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveConfigWriteTarget(flags.options(g))
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), target.Path)
			return err
		},
	}
	flags.bind(cmd)
	return cmd
}

func newConfigPathsCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "paths",
		Short: "Print user and project config paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			userPath := config.GlobalConfigPath()
			projectPath := "unavailable"
			cwd := g.cwd
			if cwd == "" {
				var err error
				cwd, err = os.Getwd()
				if err != nil {
					return apperr.Wrap(apperr.IO, "config.paths", "cwd", err)
				}
			}
			if root, err := project.FindRoot(cwd); err == nil {
				projectPath = filepath.Join(root, "m.jsonc")
			}
			r := g.mustStaticRenderer(cmd)
			return writeStaticOut(cmd, r.KeyValues([]presentation.KeyValue{
				{Key: "User", Value: userPath, Style: presentation.ValuePath},
				{Key: "Project", Value: projectPath, Style: presentation.ValuePath},
			}))
		},
	}
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
			r := g.mustStaticRenderer(cmd)
			entries := config.List(eff)
			cols := []presentation.TableColumn{
				{Key: "key", Header: "KEY", MinWidth: 8, Prefer: 28, Primary: true, Truncate: presentation.TruncateMiddle},
				{Key: "value", Header: "VALUE", MinWidth: 4, Prefer: 32, Truncate: presentation.TruncateMiddle},
				{Key: "values", Header: "VALUES", MinWidth: 3, Prefer: 36, Truncate: presentation.TruncateMiddle},
			}
			if sources {
				cols = append(cols,
					presentation.TableColumn{Key: "source", Header: "SOURCE", MinWidth: 4, Prefer: 12},
					presentation.TableColumn{Key: "path", Header: "PATH", MinWidth: 4, Prefer: 24, Truncate: presentation.TruncateMiddle},
				)
			}
			rows := make([]map[string]string, 0, len(entries))
			for _, e := range entries {
				values := e.Values
				if values == "" {
					values = "-"
				}
				row := map[string]string{
					"key":    e.Key,
					"value":  diagnostics.Redact(e.Value),
					"values": values,
				}
				if sources {
					row["source"] = displayConfigSource(e.Source)
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

func writeConfigMutationResult(g *globalFlags, cmd *cobra.Command, verb, key, value string, target configWriteTarget) error {
	if configOutputQuiet(g, cmd) {
		return nil
	}
	r := g.mustStaticRenderer(cmd)
	var headline string
	if value == "" {
		headline = fmt.Sprintf("%s %s", verb, key)
	} else {
		headline = fmt.Sprintf("%s %s = %s", verb, key, diagnostics.Redact(value))
	}
	body := r.KeyValues([]presentation.KeyValue{
		{Key: "Scope", Value: string(target.Scope)},
		{Key: "Path", Value: target.Path, Style: presentation.ValuePath},
	})
	return writeStaticOut(cmd, headline+"\n\n"+body)
}

func configOutputStructured(g *globalFlags, cmd *cobra.Command) bool {
	ctrl, err := g.controller(cmd)
	if err != nil {
		return false
	}
	return ctrl.Options().Structured()
}

func configOutputQuiet(g *globalFlags, cmd *cobra.Command) bool {
	ctrl, err := g.controller(cmd)
	if err != nil {
		return false
	}
	opts := ctrl.Options()
	return opts.Structured() || opts.Output == presentation.OutputSilent
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

// checkUserScopedKey rejects project-scoped writes for keys that are user-only.
func checkUserScopedKey(key string, target configWriteTarget) error {
	if !userScopedKeys[key] {
		return nil
	}
	if target.Scope == configWriteProject {
		return apperr.New(apperr.Usage, "config.set", key,
			key+" is a user-scoped setting; cannot write to project config")
	}
	return nil
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
