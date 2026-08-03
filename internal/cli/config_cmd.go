package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/presentation"
)

func newConfigCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `Manage Mew configuration.`,
	}
	cmd.AddCommand(newConfigGetCmd(g))
	cmd.AddCommand(newConfigSetCmd(g))
	cmd.AddCommand(newConfigUnsetCmd(g))
	cmd.AddCommand(newConfigListCmd(g))
	cmd.AddCommand(newConfigExplainCmd(g))
	cmd.AddCommand(newConfigEditCmd(g))
	cmd.AddCommand(newConfigPathCmd(g))
	cmd.AddCommand(newConfigValidateCmd(g))
	cmd.AddCommand(newConfigMigrateCmd(g))
	cmd.AddCommand(newConfigResetCmd(g))
	return cmd
}

// ── get ───────────────────────────────────────────────────────

func newConfigGetCmd(g *globalFlags) *cobra.Command {
	var (
		flags   configWriteFlags
		verbose bool
	)
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Print a config value (default: user scope)",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeConfigKeys(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if err := flags.validateScope(); err != nil {
				return err
			}
			eff, err := invocationConfig(g)
			if err != nil {
				return err
			}
			scope := flags.resolvedScope()
			view, err := resolveConfigGet(eff, key, scope)
			if err != nil {
				if configOutputStructured(g, cmd) {
					// Structured consumers get the shape they expect even for a
					// key the scope does not declare; the typed error still sets
					// the exit code.
					if nse := (*notSetError)(nil); errors.As(err, &nse) {
						_ = writeConfigJSON(cmd, configGetNotSetJSON(eff, nse.key, scope))
					}
				}
				return err
			}

			if configOutputStructured(g, cmd) {
				return writeConfigJSON(cmd, view.json())
			}

			if !verbose {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), view.Entry.Value)
				return err
			}
			return writeStaticOut(cmd, renderConfigGetVerbose(g, cmd, view))
		},
	}
	flags.bind(cmd)
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show full metadata")
	return cmd
}

// configGetNotSetJSON is the structured document for a key that is registered
// but unset at the requested raw scope. Reporting configured=false before the
// typed error keeps the JSON shape stable for consumers that parse stdout.
func configGetNotSetJSON(eff *config.Effective, key string, scope configScope) configGetJSON {
	out := configGetJSON{
		Key:        key,
		Scope:      string(scope),
		Configured: false,
		IsSecret:   config.IsSecret(key),
	}
	if spec := config.KeySpec(key); spec != nil {
		out.Type = string(spec.Type)
	}
	if ev, err := config.GetEffective(eff, key); err == nil {
		out.EffectiveValue = config.RedactValue(key, ev.Raw)
		out.Source = displayConfigSource(ev.Source)
		out.IsDefault = ev.Source == config.SourceDefaults
	}
	return out
}

// renderConfigGetVerbose renders the metadata block for `config get -v`.
// Only semantically available fields appear.
func renderConfigGetVerbose(g *globalFlags, cmd *cobra.Command, view configGetView) string {
	r := g.mustStaticRenderer(cmd)
	kvs := []presentation.KeyValue{
		{Key: "Key", Value: view.Entry.Key},
	}
	if view.Entry.Configured {
		kvs = append(kvs, presentation.KeyValue{Key: "Configured", Value: view.Entry.Value})
	}
	if view.EffectiveKnown {
		kvs = append(kvs, presentation.KeyValue{Key: "Effective", Value: view.EffectiveValue})
	}
	kvs = append(kvs,
		presentation.KeyValue{Key: "Scope", Value: string(view.Entry.Scope)},
		presentation.KeyValue{Key: "Source", Value: view.Entry.Source},
	)
	if view.Entry.Path != "" {
		kvs = append(kvs, presentation.KeyValue{Key: "File", Value: view.Entry.Path, Style: presentation.ValuePath})
	}
	if spec := view.Spec; spec != nil {
		typeStr := string(spec.Type)
		if spec.Type == config.TypeEnum {
			typeStr = "enum"
		}
		kvs = append(kvs,
			presentation.KeyValue{Key: "Type", Value: typeStr},
			presentation.KeyValue{Key: "Default", Value: config.RedactString(view.Entry.Key, formatConfigValue(spec.Default))},
		)
		if len(spec.Enum) > 0 {
			kvs = append(kvs, presentation.KeyValue{Key: "Allowed", Value: strings.Join(spec.Enum, ", ")})
		}
	}
	if view.Entry.IsSecret {
		kvs = append(kvs, presentation.KeyValue{Key: "Is secret", Value: "true"})
	}
	return "\n" + r.KeyValues(kvs)
}

// ── set ───────────────────────────────────────────────────────

func newConfigSetCmd(g *globalFlags) *cobra.Command {
	var flags configWriteFlags
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config key (default: user scope)",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeConfigKeys(toComplete), cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 1 {
				return completeConfigEnumValues(args[0]), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key, raw := args[0], args[1]
			if err := flags.validateWritable(); err != nil {
				return err
			}
			val, err := config.ParseValue(key, raw)
			if err != nil {
				return apperr.Wrap(apperr.Usage, "config.set", key, err)
			}
			scope := flags.resolvedScope()
			target, err := resolveConfigWriteTarget(scope, g.cwd)
			if err != nil {
				return err
			}
			if err := checkWritableScope(key, target.Scope); err != nil {
				return err
			}

			// Previous is the value this scope held, read from its own layer.
			// The effective winner may come from a different layer entirely, and
			// reporting that as "previous" would describe a value the write did
			// not replace.
			eff, err := invocationConfig(g)
			if err != nil {
				return err
			}
			prev, prevSet := scopeValueOrUnset(eff, scope, key)

			if err := config.SetFile(target.Path, key, val); err != nil {
				return err
			}

			// One reload republishes the invocation snapshot so the reported
			// target-scope and effective values both come from the same state.
			reloaded, reloadErr := reloadInvocationConfig(cmd.Context(), g)
			current := config.RedactString(key, formatConfigValue(val))
			effective := ""
			if reloadErr == nil {
				if cur, ok := scopeValueOrUnset(reloaded, scope, key); ok {
					current = cur
				}
				if ev, eerr := config.GetEffective(reloaded, canonicalConfigKey(key)); eerr == nil {
					effective = config.RedactString(key, formatConfigValue(ev.Raw))
				}
			}
			return writeConfigSetResult(g, cmd, key, prev, prevSet, current, effective, target)
		},
	}
	flags.bind(cmd)
	return cmd
}

func writeConfigSetResult(g *globalFlags, cmd *cobra.Command, key, prev string, prevSet bool, current, effective string, target configWriteTarget) error {
	if configOutputQuiet(g, cmd) {
		return nil
	}
	key = canonicalConfigKey(key)
	r := g.mustStaticRenderer(cmd)
	headline := fmt.Sprintf("%s Updated %s", r.Settings().Symbols.Success, key)
	prevDisplay := prev
	if !prevSet {
		prevDisplay = "(unset)"
	}
	kvs := []presentation.KeyValue{
		{Key: "Previous", Value: prevDisplay},
		{Key: "Current", Value: current},
		{Key: "Scope", Value: string(target.Scope)},
	}
	if effective != "" && effective != current {
		kvs = append(kvs, presentation.KeyValue{Key: "Effective", Value: effective})
	}
	kvs = append(kvs, presentation.KeyValue{Key: "File", Value: target.Path, Style: presentation.ValuePath})
	return writeStaticOut(cmd, headline+"\n\n"+r.KeyValues(kvs))
}

// ── unset ─────────────────────────────────────────────────────

func newConfigUnsetCmd(g *globalFlags) *cobra.Command {
	var flags configWriteFlags
	cmd := &cobra.Command{
		Use:     "unset <key>",
		Aliases: []string{"rm"},
		Short:   "Remove a config key (default: user scope)",
		Args:    cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeConfigKeys(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if err := flags.validateWritable(); err != nil {
				return err
			}
			scope := flags.resolvedScope()
			target, err := resolveConfigWriteTarget(scope, g.cwd)
			if err != nil {
				return err
			}
			if err := checkWritableScope(key, target.Scope); err != nil {
				return err
			}
			// UnsetFile writes nothing when the key is absent, so an already
			// unset key is an idempotent no-op and other layers are untouched.
			if err := config.UnsetFile(target.Path, key); err != nil {
				return err
			}
			return writeConfigUnsetResult(cmd.Context(), g, cmd, key, scope, target.Path)
		},
	}
	flags.bind(cmd)
	return cmd
}

func writeConfigUnsetResult(ctx context.Context, g *globalFlags, cmd *cobra.Command, key string, scope configScope, path string) error {
	if configOutputQuiet(g, cmd) {
		return nil
	}
	canon := canonicalConfigKey(key)
	// One reload, after the write, so the reported fallback is the layer that
	// actually wins now rather than the one that won before.
	var fallback, fallbackSrc string
	if eff, err := reloadInvocationConfig(ctx, g); err == nil {
		if v, gerr := config.GetEffective(eff, canon); gerr == nil {
			fallback = config.RedactString(canon, formatConfigValue(v.Raw))
			fallbackSrc = displayConfigSource(v.Source)
		}
	}

	r := g.mustStaticRenderer(cmd)
	headline := fmt.Sprintf("%s Removed %s from %s configuration",
		r.Settings().Symbols.Success, canon, scope)
	kvs := make([]presentation.KeyValue, 0, 3)
	if fallback != "" {
		kvs = append(kvs, presentation.KeyValue{Key: "Effective", Value: fallback})
	}
	if fallbackSrc != "" {
		kvs = append(kvs, presentation.KeyValue{Key: "Source", Value: fallbackSrc})
	}
	kvs = append(kvs, presentation.KeyValue{Key: "File", Value: path, Style: presentation.ValuePath})
	return writeStaticOut(cmd, headline+"\n\n"+r.KeyValues(kvs))
}

// ── list ──────────────────────────────────────────────────────

func newConfigListCmd(g *globalFlags) *cobra.Command {
	var (
		flags        configWriteFlags
		showOrigin   bool
		changed      bool
		inclDefaults bool
		prefix       string
	)
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configuration",
		Long:    "List configuration values. Default scope is user.",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := flags.validateScope(); err != nil {
				return err
			}
			scope := flags.resolvedScope()
			eff, err := invocationConfig(g)
			if err != nil {
				return err
			}
			opts := configListOptions{prefix: prefix, changed: changed, inclDefaults: inclDefaults}
			entries := resolveConfigList(eff, scope, opts)

			if configOutputStructured(g, cmd) {
				return writeConfigListJSON(cmd, entries, scope, showOrigin)
			}
			return writeConfigListHuman(g, cmd, entries, scope, showOrigin)
		},
	}
	flags.bind(cmd)
	cmd.Flags().BoolVar(&showOrigin, "show-origin", false, "show value source and file")
	cmd.Flags().BoolVar(&changed, "changed", false, "show only values different from defaults")
	cmd.Flags().BoolVar(&inclDefaults, "defaults", false, "include registered schema defaults")
	cmd.Flags().StringVar(&prefix, "prefix", "", "filter keys by namespace prefix")
	return cmd
}

func writeConfigListHuman(g *globalFlags, cmd *cobra.Command, entries []configEntryView, scope configScope, showOrigin bool) error {
	r := g.mustStaticRenderer(cmd)
	settings := r.Settings()
	// Narrow terminals and accessible mode get one field per line; the same
	// threshold the shared KeyValues renderer uses.
	stacked := settings.Width < 60 || settings.Accessible

	var b strings.Builder
	b.WriteString(configScopeLabel(scope))
	b.WriteString("\n\n")

	var configured, defaulted int
	lastGroup := ""
	keyWidth := 0
	for _, e := range entries {
		if w := presentation.CellWidth(e.Key); w > keyWidth {
			keyWidth = w
		}
	}
	for _, e := range entries {
		if e.Configured {
			configured++
		} else {
			defaulted++
		}
		if e.Group != lastGroup {
			if lastGroup != "" {
				b.WriteString("\n")
			}
			group := e.Group
			if group == "" {
				group = "Other"
			}
			b.WriteString(group)
			b.WriteString("\n")
			lastGroup = e.Group
		}
		b.WriteString(configListRow(e, keyWidth, showOrigin, stacked))
		b.WriteString("\n")
	}
	if len(entries) > 0 {
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("%d configured, %d defaults\n", configured, defaulted))
	return writeStaticOut(cmd, b.String())
}

// configListRow renders one list row, padded to keyWidth so columns align on
// visible width rather than byte length.
func configListRow(e configEntryView, keyWidth int, showOrigin, stacked bool) string {
	if stacked {
		var b strings.Builder
		b.WriteString("  ")
		b.WriteString(e.Key)
		b.WriteString("\n    ")
		b.WriteString(e.Value)
		if showOrigin {
			b.WriteString("\n    ")
			b.WriteString(e.Source)
			if e.Path != "" {
				b.WriteString(" ")
				b.WriteString(e.Path)
			}
		}
		return b.String()
	}
	pad := keyWidth - presentation.CellWidth(e.Key)
	if pad < 0 {
		pad = 0
	}
	line := "  " + e.Key + strings.Repeat(" ", pad+2) + e.Value
	if showOrigin {
		line += "  [" + e.Source + "]"
		if e.Path != "" {
			line += " " + e.Path
		}
	}
	return line
}

func configScopeLabel(scope configScope) string {
	switch scope {
	case configScopeProject:
		return "Project configuration"
	case configScopeEffective:
		return "Effective configuration"
	default:
		return "User configuration"
	}
}

func writeConfigListJSON(cmd *cobra.Command, entries []configEntryView, scope configScope, showOrigin bool) error {
	rows := make([]configEntryJSON, 0, len(entries))
	for _, e := range entries {
		row := e.json()
		if !showOrigin {
			// Provenance detail is opt-in; the source name itself always stays.
			row.Path = ""
		}
		rows = append(rows, row)
	}
	return writeConfigJSON(cmd, map[string]any{
		"scope":   string(scope),
		"entries": rows,
	})
}

// ── explain ───────────────────────────────────────────────────

func newConfigExplainCmd(g *globalFlags) *cobra.Command {
	var flags configWriteFlags
	cmd := &cobra.Command{
		Use:   "explain <key>",
		Short: "Show config key resolution chain",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeConfigKeys(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if err := flags.validateScope(); err != nil {
				return err
			}
			eff, err := invocationConfig(g)
			if err != nil {
				return err
			}
			view, err := resolveConfigExplain(eff, key, flags.resolvedScope())
			if err != nil {
				return err
			}

			if configOutputStructured(g, cmd) {
				return writeConfigJSON(cmd, view.json())
			}
			return writeStaticOut(cmd, renderConfigExplainHuman(g, cmd, view))
		},
	}
	flags.bind(cmd)
	return cmd
}

func renderConfigExplainHuman(g *globalFlags, cmd *cobra.Command, view configResolutionView) string {
	r := g.mustStaticRenderer(cmd)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s = %s\n", view.Key, view.Effective.Value))
	if view.Spec != nil && view.Spec.Description != "" {
		b.WriteString(view.Spec.Description)
		b.WriteString("\n")
	}
	b.WriteString("\nResolution\n")

	// Layer rows align on the widest source name so the effective marker lines
	// up regardless of which layers are present.
	srcWidth := 0
	for _, l := range view.Layers {
		if w := presentation.CellWidth(l.Source); w > srcWidth {
			srcWidth = w
		}
	}
	for _, l := range view.Layers {
		pad := srcWidth - presentation.CellWidth(l.Source)
		if pad < 0 {
			pad = 0
		}
		b.WriteString("  " + l.Source + strings.Repeat(" ", pad+2) + l.Value)
		if l.Effective {
			b.WriteString("  <- effective")
		}
		b.WriteString("\n")
	}

	if view.Selected != nil {
		// Labelled so the requested-scope value cannot be mistaken for another
		// rung of the resolution chain above.
		b.WriteString("\nRequested scope\n")
		b.WriteString(r.KeyValues([]presentation.KeyValue{
			{Key: string(view.Selected.Scope), Value: view.Selected.Value},
		}))
		b.WriteString("\n")
	}

	if spec := view.Spec; spec != nil {
		b.WriteString("\nSchema\n")
		typeStr := string(spec.Type)
		if spec.Type == config.TypeEnum {
			typeStr = "enum"
		}
		kv := []presentation.KeyValue{
			{Key: "Type", Value: typeStr},
			{Key: "Default", Value: config.RedactString(view.Key, formatConfigValue(spec.Default))},
		}
		if len(spec.Enum) > 0 {
			kv = append(kv, presentation.KeyValue{Key: "Allowed", Value: strings.Join(spec.Enum, ", ")})
		}
		if rng := configRangeText(spec); rng != "" {
			kv = append(kv, presentation.KeyValue{Key: "Range", Value: rng})
		}
		if scopes := configScopeNames(spec); len(scopes) > 0 {
			kv = append(kv, presentation.KeyValue{Key: "Scopes", Value: strings.Join(scopes, ", ")})
		}
		if len(spec.Commands) > 0 {
			kv = append(kv, presentation.KeyValue{Key: "Used by", Value: strings.Join(spec.Commands, ", ")})
		}
		if env := config.EnvVar(view.Key); env != "" {
			kv = append(kv, presentation.KeyValue{Key: "Env var", Value: env})
		}
		if view.LegacyKey != "" {
			kv = append(kv, presentation.KeyValue{Key: "Legacy key", Value: view.LegacyKey})
		}
		if spec.Deprecated {
			kv = append(kv, presentation.KeyValue{Key: "Deprecated", Value: "true"})
		}
		if spec.Replacement != "" {
			kv = append(kv, presentation.KeyValue{Key: "Replaced by", Value: spec.Replacement})
		}
		if spec.Secret {
			kv = append(kv, presentation.KeyValue{Key: "Is secret", Value: "true"})
		}
		b.WriteString(r.KeyValues(kv))
		b.WriteString("\n")
	}
	return b.String()
}

// configRangeText renders an int key's min/max bounds, or "" when unbounded.
func configRangeText(spec *config.ConfigKeySpec) string {
	if spec.Minimum == nil && spec.Maximum == nil {
		return ""
	}
	switch {
	case spec.Minimum != nil && spec.Maximum != nil:
		return fmt.Sprintf("%d..%d", *spec.Minimum, *spec.Maximum)
	case spec.Minimum != nil:
		return fmt.Sprintf(">= %d", *spec.Minimum)
	default:
		return fmt.Sprintf("<= %d", *spec.Maximum)
	}
}

// ── edit ──────────────────────────────────────────────────────

func newConfigEditCmd(g *globalFlags) *cobra.Command {
	var flags configWriteFlags
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Open config file in editor (default: user scope)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := flags.validateWritable(); err != nil {
				return err
			}
			scope := flags.resolvedScope()
			cwd := g.cwd
			target, err := resolveConfigWriteTarget(scope, cwd)
			if err != nil {
				return err
			}

			// Create file if missing.
			if _, err := os.Stat(target.Path); os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(target.Path), 0o755); err != nil {
					return apperr.Wrap(apperr.IO, "config.edit", target.Path, err)
				}
				if err := os.WriteFile(target.Path, []byte("{}\n"), 0o644); err != nil {
					return apperr.Wrap(apperr.IO, "config.edit", target.Path, err)
				}
			}

			// Read existing content for recovery.
			orig, err := os.ReadFile(target.Path)
			if err != nil {
				return apperr.Wrap(apperr.IO, "config.edit", target.Path, err)
			}

			editor := resolveEditor()
			ecmd := exec.Command(editor, target.Path)
			ecmd.Stdin = os.Stdin
			ecmd.Stdout = os.Stdout
			ecmd.Stderr = os.Stderr
			if err := ecmd.Run(); err != nil {
				return apperr.Wrap(apperr.IO, "config.edit", editor, err)
			}

			// Validate after edit.
			b, err := os.ReadFile(target.Path)
			if err != nil {
				return apperr.Wrap(apperr.IO, "config.edit", target.Path, err)
			}
			if _, err := config.ParseJSONC(b); err != nil {
				// Restore original.
				if writeErr := os.WriteFile(target.Path, orig, 0o644); writeErr != nil {
					return apperr.Wrap(apperr.IO, "config.edit", target.Path,
						fmt.Errorf("invalid config (%w) and failed to restore backup (%w)", err, writeErr))
				}
				return apperr.Wrap(apperr.Config, "config.edit", target.Path,
					fmt.Errorf("invalid config, original restored: %w", err))
			}

			r := g.mustStaticRenderer(cmd)
			return writeStaticOut(cmd, "✓ Config saved\n\n"+r.KeyValues([]presentation.KeyValue{
				{Key: "Scope", Value: string(target.Scope)},
				{Key: "File", Value: target.Path, Style: presentation.ValuePath},
			}))
		},
	}
	flags.bind(cmd)
	return cmd
}

func resolveEditor() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	return "notepad"
}

// ── path ──────────────────────────────────────────────────────

func newConfigPathCmd(g *globalFlags) *cobra.Command {
	var (
		flags configWriteFlags
		all   bool
	)
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Print config file path (default: user scope)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if configOutputStructured(g, cmd) {
				return writeConfigPathJSON(cmd, g.cwd, flags.resolvedScope(), all)
			}

			cwd := g.cwd

			if all {
				userPath, projPath := resolveEffectivePaths(cwd)
				r := g.mustStaticRenderer(cmd)
				kv := []presentation.KeyValue{
					{Key: "User", Value: userPath, Style: presentation.ValuePath},
				}
				if projPath != "" {
					kv = append(kv, presentation.KeyValue{Key: "Project", Value: projPath, Style: presentation.ValuePath})
				} else {
					kv = append(kv, presentation.KeyValue{Key: "Project", Value: "unavailable"})
				}
				return writeStaticOut(cmd, r.KeyValues(kv))
			}

			scope := flags.resolvedScope()
			if scope == configScopeEffective {
				userPath, projPath := resolveEffectivePaths(cwd)
				r := g.mustStaticRenderer(cmd)
				kv := []presentation.KeyValue{
					{Key: "User", Value: userPath, Style: presentation.ValuePath},
				}
				if projPath != "" {
					kv = append(kv, presentation.KeyValue{Key: "Project", Value: projPath, Style: presentation.ValuePath})
				}
				return writeStaticOut(cmd, r.KeyValues(kv))
			}

			target, err := resolveConfigWriteTarget(scope, cwd)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), target.Path)
			return err
		},
	}
	flags.bind(cmd)
	cmd.Flags().BoolVar(&all, "all", false, "show all config file paths")
	return cmd
}

func writeConfigPathJSON(cmd *cobra.Command, cwd string, scope configScope, all bool) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetEscapeHTML(false)
	userPath, projPath := resolveEffectivePaths(cwd)
	out := map[string]any{
		"user":    userPath,
		"project": projPath,
		"scope":   string(scope),
	}
	_ = all
	return enc.Encode(out)
}

// ── validate ──────────────────────────────────────────────────

func newConfigValidateCmd(g *globalFlags) *cobra.Command {
	var (
		flags  configWriteFlags
		strict bool
	)
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate config file (default: user scope)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := flags.validateScope(); err != nil {
				return err
			}
			scope := flags.resolvedScope()
			cwd := g.cwd

			var files []string
			switch scope {
			case configScopeUser:
				files = []string{config.GlobalConfigPath()}
			case configScopeProject:
				target, err := resolveConfigWriteTarget(configScopeProject, cwd)
				if err != nil {
					return err
				}
				files = []string{target.Path}
			case configScopeEffective:
				userPath, projPath := resolveEffectivePaths(cwd)
				files = []string{userPath}
				if projPath != "" {
					files = append(files, projPath)
				}
			}

			var allErrors []string
			var allWarnings []string
			totalKeys := 0
			type fileResult struct {
				path     string
				errors   []string
				warnings []string
				keys     int
			}
			var results []fileResult

			for _, f := range files {
				if _, err := os.Stat(f); os.IsNotExist(err) {
					continue
				}
				b, err := os.ReadFile(f)
				if err != nil {
					allErrors = append(allErrors, fmt.Sprintf("Cannot read %s: %v", f, err))
					continue
				}
				parsed, err := config.ParseJSONC(b)
				if err != nil {
					fr := fileResult{path: f}
					fr.errors = append(fr.errors, fmt.Sprintf("Parse error: %v", err))
					results = append(results, fr)
					continue
				}
				m, ok := parsed.(map[string]any)
				if !ok {
					results = append(results, fileResult{path: f, errors: []string{"Root must be an object"}})
					continue
				}
				fr := fileResult{path: f}
				for k := range flattenMap(m, "") {
					fr.keys++
					canon, isLegacy, known := resolveKeyLocal(k)
					if !known {
						fr.errors = append(fr.errors, fmt.Sprintf("Unknown key %q", k))
						continue
					}
					if isLegacy {
						msg := fmt.Sprintf("%s: use %s", k, canon)
						if strict {
							fr.errors = append(fr.errors, msg)
						} else {
							fr.warnings = append(fr.warnings, msg)
						}
					}
					if canon != "" {
						if _, exists := m[canon]; exists && isLegacy {
							fr.errors = append(fr.errors,
								fmt.Sprintf("Conflicting keys %q and %q; remove one", k, canon))
						}
					}
				}
				results = append(results, fr)
				totalKeys += fr.keys
				allErrors = append(allErrors, fr.errors...)
				allWarnings = append(allWarnings, fr.warnings...)
			}

			if configOutputStructured(g, cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(map[string]any{
					"scope":    string(scope),
					"valid":    len(allErrors) == 0,
					"keys":     totalKeys,
					"errors":   allErrors,
					"warnings": allWarnings,
					"files":    files,
				})
			}

			r := g.mustStaticRenderer(cmd)
			label := fmt.Sprintf("%s configuration", strings.ToUpper(string(scope)[:1])+string(scope)[1:])
			if len(allErrors) == 0 {
				var b strings.Builder
				b.WriteString(fmt.Sprintf("✓ %s is valid\n\n", label))
				b.WriteString(r.KeyValues([]presentation.KeyValue{
					{Key: "Keys", Value: strconv.Itoa(totalKeys)},
				}))
				if len(files) > 0 {
					b.WriteString(r.KeyValues([]presentation.KeyValue{
						{Key: "File", Value: strings.Join(files, "\n"), Style: presentation.ValuePath},
					}))
				}
				if len(allWarnings) > 0 {
					b.WriteString("\nWarnings:\n")
					for _, w := range allWarnings {
						b.WriteString("  ")
						b.WriteString(w)
						b.WriteString("\n")
					}
				}
				return writeStaticOut(cmd, b.String())
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("× %s is invalid\n\n", label))
			for _, fr := range results {
				for _, e := range fr.errors {
					b.WriteString(fmt.Sprintf("  %s\n  File %s\n\n", e, fr.path))
				}
			}
			if len(allWarnings) > 0 {
				b.WriteString("Warnings:\n")
				for _, w := range allWarnings {
					b.WriteString(fmt.Sprintf("  %s\n", w))
				}
			}
			return writeStaticOut(cmd, b.String())
		},
	}
	flags.bind(cmd)
	cmd.Flags().BoolVar(&strict, "strict", false, "treat legacy keys as errors")
	return cmd
}

func resolveKeyLocal(key string) (canonical string, isLegacy bool, known bool) {
	canon := config.CanonicalKey(key)
	if canon != "" && canon != key {
		return canon, true, true
	}
	if config.IsCanonical(key) {
		return key, false, true
	}
	return "", false, false
}

func flattenMap(m map[string]any, prefix string) map[string]any {
	out := map[string]any{}
	for k, child := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if cm, ok := child.(map[string]any); ok {
			for ck, cv := range flattenMap(cm, key) {
				out[ck] = cv
			}
		} else {
			out[key] = child
		}
	}
	return out
}

// ── migrate ───────────────────────────────────────────────────

func newConfigMigrateCmd(g *globalFlags) *cobra.Command {
	var (
		flags configWriteFlags
		check bool
	)
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate legacy config keys to canonical snake_case (default: user scope)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := flags.validateWritable(); err != nil {
				return err
			}
			scope := flags.resolvedScope()
			cwd := g.cwd
			target, err := resolveConfigWriteTarget(scope, cwd)
			if err != nil {
				return err
			}

			if check {
				needed, err := config.CheckMigration(target.Path)
				if err != nil {
					return err
				}
				if configOutputStructured(g, cmd) {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetEscapeHTML(false)
					return enc.Encode(map[string]any{
						"needs_migration": len(needed) > 0,
						"changes":         needed,
						"file":            target.Path,
					})
				}
				if len(needed) == 0 {
					return writeStaticOut(cmd, "Already canonical.")
				}
				var b strings.Builder
				b.WriteString("! Configuration uses deprecated keys\n\n")
				for legacy, canon := range needed {
					b.WriteString(fmt.Sprintf("  %s\n    Use %s\n\n", legacy, canon))
				}
				b.WriteString("Run `m config migrate` to update the file.\n")
				return writeStaticOut(cmd, b.String())
			}

			count, err := config.MigrateFile(target.Path)
			if err != nil {
				return err
			}
			if configOutputStructured(g, cmd) {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(map[string]any{
					"migrated": count,
					"file":     target.Path,
				})
			}
			r := g.mustStaticRenderer(cmd)
			if count == 0 {
				return writeStaticOut(cmd, "Already canonical.")
			}
			return writeStaticOut(cmd, fmt.Sprintf("✓ Migrated %d key(s) in %s\n\n%s",
				count, target.Path,
				r.KeyValues([]presentation.KeyValue{
					{Key: "Scope", Value: string(target.Scope)},
					{Key: "File", Value: target.Path, Style: presentation.ValuePath},
				})))
		},
	}
	flags.bind(cmd)
	cmd.Flags().BoolVar(&check, "check", false, "dry-run: report what would change without writing")
	return cmd
}

// ── reset ─────────────────────────────────────────────────────

func newConfigResetCmd(g *globalFlags) *cobra.Command {
	var (
		flags configWriteFlags
		yes   bool
	)
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset config to defaults (default: user scope)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := flags.validateWritable(); err != nil {
				return err
			}
			scope := flags.resolvedScope()
			cwd := g.cwd
			target, err := resolveConfigWriteTarget(scope, cwd)
			if err != nil {
				return err
			}

			if _, err := os.Stat(target.Path); os.IsNotExist(err) {
				return writeStaticOut(cmd,
					fmt.Sprintf("No %s config file to reset.\n\n%s",
						scope, target.Path))
			}

			if !yes {
				return apperr.New(apperr.Usage, "config.reset", string(scope),
					"use --yes to confirm reset")
			}

			if err := os.Remove(target.Path); err != nil {
				return apperr.Wrap(apperr.IO, "config.reset", target.Path, err)
			}
			r := g.mustStaticRenderer(cmd)
			return writeStaticOut(cmd, fmt.Sprintf("✓ Reset %s configuration\n\n%s",
				scope,
				r.KeyValues([]presentation.KeyValue{
					{Key: "File", Value: target.Path, Style: presentation.ValuePath},
				})))
		},
	}
	flags.bind(cmd)
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

// ── completion helpers ────────────────────────────────────────

func completeConfigKeys(toComplete string) []string {
	var out []string
	for _, k := range config.RegisteredKeys() {
		if strings.HasPrefix(k, toComplete) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func completeConfigEnumValues(key string) []string {
	spec := config.KeySpec(key)
	if spec == nil || len(spec.Enum) == 0 {
		return nil
	}
	return spec.Enum
}

// ── helpers ───────────────────────────────────────────────────

// invocationConfig returns the one configuration snapshot bootstrap resolved
// for this invocation. Config commands never load a second time: --config, the
// project root, and the environment were all interpreted once, and reading them
// again here could disagree with what the rest of the invocation sees.
//
// A nil snapshot means the command was driven without going through the
// bootstrap path, which only happens in tests; loading once on that path keeps
// them working without giving production a second loader.
func invocationConfig(g *globalFlags) (*config.Effective, error) {
	if g != nil && g.snapshot != nil && g.snapshot.Config != nil {
		return g.snapshot.Config, nil
	}
	return reloadInvocationConfig(context.Background(), g)
}

// reloadInvocationConfig republishes the invocation snapshot from the same
// inputs bootstrap used, and returns the fresh effective config. Write commands
// call it exactly once after mutating a file so every value they report comes
// from one post-write state.
func reloadInvocationConfig(ctx context.Context, g *globalFlags) (*config.Effective, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	snap, err := loadConfigFn(ctx, app.Options{
		CWD:           g.cwd,
		ConfigPath:    g.configPath,
		Offline:       g.offline,
		PreferOffline: g.preferOffline,
	})
	if err != nil {
		return nil, err
	}
	g.snapshot = &snap
	g.cwd = snap.CWD
	return snap.Config, nil
}

// canonicalConfigKey returns the canonical spelling of key, or key itself when
// it is not a registered or legacy name.
func canonicalConfigKey(key string) string {
	if c := config.CanonicalKey(key); c != "" {
		return c
	}
	return key
}

// scopeValueOrUnset returns the display value a raw scope holds for key, and
// whether the scope declares it at all. Callers report "unset" rather than an
// empty string so a missing key never reads as an empty value.
func scopeValueOrUnset(eff *config.Effective, scope configScope, key string) (string, bool) {
	canon := canonicalConfigKey(key)
	v, err := config.GetAtScope(eff, configScopeToConfig(scope), canon)
	if err != nil {
		return "", false
	}
	return config.RedactString(canon, formatConfigValue(v.Raw)), true
}

// checkWritableScope enforces the schema's per-key writable scopes. The scope
// list comes from ConfigKeySpec so the CLI keeps no second copy of the policy.
// Unknown Mew-owned keys are rejected earlier by ParseValue and UnsetFile;
// dynamic registries.* keys have no spec and keep their existing any-scope
// policy.
func checkWritableScope(key string, scope configScope) error {
	spec := config.KeySpec(canonicalConfigKey(key))
	if spec == nil {
		return nil
	}
	target := configScopeToConfig(scope)
	for _, s := range spec.Scopes {
		if s == target {
			return nil
		}
	}
	if len(spec.Scopes) == 0 {
		return nil
	}
	return apperr.New(apperr.Usage, "config.write", spec.Key,
		fmt.Sprintf("%s is writable in %s scope; cannot write to %s config",
			spec.Key, strings.Join(configScopeNames(spec), ", "), scope))
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
