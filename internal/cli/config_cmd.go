package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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
			eff, err := loadEffective(g)
			if err != nil {
				return err
			}
			scope := flags.resolvedScope()
			v, err := config.Get(eff, key)
			if err != nil {
				return err
			}

			if configOutputStructured(g, cmd) {
				return writeConfigGetJSON(cmd, key, v, scope, eff)
			}

			if !verbose {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), formatConfigValue(v.Raw))
				return err
			}

			r := g.mustStaticRenderer(cmd)
			spec := config.KeySpec(key)
			lines := []string{"", r.KeyValues([]presentation.KeyValue{
				{Key: "Value", Value: diagnostics.Redact(formatConfigValue(v.Raw))},
				{Key: "Source", Value: displayConfigSource(v.Source)},
				{Key: "Path", Value: v.Path, Style: presentation.ValuePath},
			})}
			if spec != nil {
				extra := []presentation.KeyValue{
					{Key: "Description", Value: spec.Description},
				}
				typeStr := string(spec.Type)
				if spec.Type == config.TypeEnum {
					typeStr = "enum (" + strings.Join(spec.Enum, ", ") + ")"
				}
				extra = append(extra, presentation.KeyValue{Key: "Type", Value: typeStr})
				extra = append(extra, presentation.KeyValue{Key: "Default", Value: formatConfigValue(spec.Default)})
				if len(spec.Enum) > 0 {
					extra = append(extra, presentation.KeyValue{Key: "Allowed", Value: strings.Join(spec.Enum, ", ")})
				}
				lines = append(lines, r.KeyValues(extra))
			}
			return writeStaticOut(cmd, strings.Join(lines, "\n"))
		},
	}
	flags.bind(cmd)
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show full metadata")
	return cmd
}

func writeConfigGetJSON(cmd *cobra.Command, key string, v config.Value, scope configScope, eff *config.Effective) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetEscapeHTML(false)
	out := map[string]any{
		"key":    key,
		"value":  v.Raw,
		"source": displayConfigSource(v.Source),
		"path":   v.Path,
		"scope":  string(scope),
	}
	spec := config.KeySpec(key)
	if spec != nil {
		out["type"] = string(spec.Type)
		out["default"] = spec.Default
		out["is_secret"] = spec.Secret
		if spec.Secret {
			out["value"] = "<redacted>"
		}
	}
	_ = eff
	return enc.Encode(out)
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
			cwd := g.cwd
			target, err := resolveConfigWriteTarget(scope, cwd)
			if err != nil {
				return err
			}
			if err := checkUserScopedKey(key, target); err != nil {
				return err
			}
			prev := readScopeValue(target.Path, key)
			if err := config.SetFile(target.Path, key, val); err != nil {
				return err
			}
			newEffective := readEffectiveValue(g, key)
			return writeConfigSetResult(g, cmd, key, prev, formatConfigValue(val), newEffective, target)
		},
	}
	flags.bind(cmd)
	return cmd
}

func writeConfigSetResult(g *globalFlags, cmd *cobra.Command, key, prev, current, newEffective string, target configWriteTarget) error {
	if configOutputQuiet(g, cmd) {
		return nil
	}
	canon := config.CanonicalKey(key)
	if canon != "" && canon != key {
		key = canon
	}
	r := g.mustStaticRenderer(cmd)
	headline := fmt.Sprintf("✓ Updated %s", key)
	kvs := []presentation.KeyValue{
		{Key: "Previous", Value: diagnostics.Redact(prev)},
		{Key: "Current", Value: diagnostics.Redact(current)},
		{Key: "Scope", Value: string(target.Scope)},
		{Key: "File", Value: target.Path, Style: presentation.ValuePath},
	}
	if newEffective != "" && newEffective != current {
		kvs = append(kvs, presentation.KeyValue{Key: "Effective", Value: diagnostics.Redact(newEffective)})
	}
	body := r.KeyValues(kvs)
	return writeStaticOut(cmd, headline+"\n\n"+body)
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
			cwd := g.cwd
			target, err := resolveConfigWriteTarget(scope, cwd)
			if err != nil {
				return err
			}
			if err := checkUserScopedKey(key, target); err != nil {
				return err
			}
			if err := config.UnsetFile(target.Path, key); err != nil {
				return err
			}
			return writeConfigUnsetResult(g, cmd, key, scope, target.Path)
		},
	}
	flags.bind(cmd)
	return cmd
}

func writeConfigUnsetResult(g *globalFlags, cmd *cobra.Command, key string, scope configScope, path string) error {
	if configOutputQuiet(g, cmd) {
		return nil
	}
	var effectiveMsg string
	if eff, err := loadEffective(g); err == nil {
		if v, err := config.Get(eff, key); err == nil {
			effectiveMsg = fmt.Sprintf("Effective value is now %s.", formatConfigValue(v.Raw))
		}
	}
	r := g.mustStaticRenderer(cmd)
	headline := fmt.Sprintf("✓ Removed %s from %s configuration", key, scope)
	body := ""
	if effectiveMsg != "" {
		body = "\n" + effectiveMsg
	}
	body += "\n" + r.KeyValues([]presentation.KeyValue{
		{Key: "File", Value: path, Style: presentation.ValuePath},
	})
	return writeStaticOut(cmd, headline+body)
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
			eff, err := loadEffective(g)
			if err != nil {
				return err
			}

			if configOutputStructured(g, cmd) {
				return writeConfigListJSON(cmd, eff, scope, showOrigin, changed, inclDefaults, prefix)
			}

			return writeConfigListHuman(g, cmd, eff, scope, showOrigin, changed, inclDefaults, prefix)
		},
	}
	flags.bind(cmd)
	cmd.Flags().BoolVar(&showOrigin, "show-origin", false, "show value source")
	cmd.Flags().BoolVar(&changed, "changed", false, "show only values different from defaults")
	cmd.Flags().BoolVar(&inclDefaults, "defaults", false, "include every schema key including defaults")
	cmd.Flags().StringVar(&prefix, "prefix", "", "filter keys by namespace prefix")
	return cmd
}

func writeConfigListHuman(g *globalFlags, cmd *cobra.Command, eff *config.Effective, scope configScope, showOrigin, changed, inclDefaults bool, prefix string) error {
	r := g.mustStaticRenderer(cmd)

	var scopeLabel string
	switch scope {
	case configScopeUser:
		scopeLabel = "User configuration"
	case configScopeProject:
		scopeLabel = "Project configuration"
	case configScopeEffective:
		scopeLabel = "Effective configuration"
	}

	type listEntry struct {
		key    string
		value  string
		origin string
		isDef  bool
		isSec  bool
	}
	groups := make(map[string][]listEntry)
	groupOrder := config.Groups()

	for _, gname := range groupOrder {
		for _, key := range config.KeysByGroup(gname) {
			if prefix != "" && !strings.HasPrefix(key, prefix) {
				continue
			}
			v, err := config.Get(eff, key)
			if err != nil {
				continue
			}
			spec := config.KeySpec(key)
			isDefault := v.Source == config.SourceDefaults

			// Filter by scope.
			if scope == configScopeUser && v.Source != config.SourceGlobal {
				if v.Source == config.SourceDefaults {
					if !inclDefaults {
						continue
					}
				} else if v.Source != config.SourceGlobal {
					continue
				}
			}
			if scope == configScopeProject && v.Source != config.SourceProject {
				if v.Source == config.SourceDefaults {
					if !inclDefaults {
						continue
					}
				} else {
					continue
				}
			}

			if changed && isDefault {
				continue
			}

			val := formatConfigValue(v.Raw)
			if spec != nil && spec.Secret {
				val = "<redacted>"
			}

			groups[gname] = append(groups[gname], listEntry{
				key: key, value: val,
				origin: displayConfigSource(v.Source),
				isDef:  isDefault,
				isSec:  spec != nil && spec.Secret,
			})
		}
	}

	var configured, defaulted int
	for _, entries := range groups {
		for _, e := range entries {
			if e.isDef {
				defaulted++
			} else {
				configured++
			}
		}
	}

	var b strings.Builder
	b.WriteString(scopeLabel)
	b.WriteString("\n\n")

	for _, gname := range groupOrder {
		entries := groups[gname]
		if len(entries) == 0 {
			continue
		}
		// Group heading: bright white via plain text.
		b.WriteString(gname)
		b.WriteString("\n")
		for _, e := range entries {
			var line string
			if e.isDef || e.isSec {
				line = fmt.Sprintf("  %-36s %s", e.key, e.value)
			} else {
				line = fmt.Sprintf("  %-36s %s", e.key, e.value)
			}
			if showOrigin {
				line += fmt.Sprintf("  [%s]", e.origin)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	summary := fmt.Sprintf("%d configured, %d defaults", configured, defaulted)
	b.WriteString(summary)
	b.WriteString("\n")

	var filePath string
	switch scope {
	case configScopeUser:
		filePath = config.GlobalConfigPath()
	case configScopeProject:
		if cwd := g.cwd; cwd != "" {
			if root, err := project.FindRoot(cwd); err == nil {
				filePath = filepath.Join(root, "m.jsonc")
			}
		}
	case configScopeEffective:
		userPath, projPath := resolveEffectivePaths(g.cwd)
		filePath = userPath
		if projPath != "" {
			filePath += "\n" + projPath
		}
	}
	if filePath != "" {
		b.WriteString(filePath)
		b.WriteString("\n")
	}

	_ = showOrigin
	_ = r
	return writeStaticOut(cmd, b.String())
}

func writeConfigListJSON(cmd *cobra.Command, eff *config.Effective, scope configScope, showOrigin, changed, inclDefaults bool, prefix string) error {
	type jsonEntry struct {
		Key       string `json:"key"`
		Value     any    `json:"value"`
		Source    string `json:"source"`
		File      string `json:"file,omitempty"`
		Type      string `json:"type"`
		IsDefault bool   `json:"is_default"`
		IsSecret  bool   `json:"is_secret"`
	}
	var entries []jsonEntry
	for _, key := range config.RegisteredKeys() {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		v, err := config.Get(eff, key)
		if err != nil {
			continue
		}
		isDef := v.Source == config.SourceDefaults
		if !inclDefaults && isDef && scope != configScopeEffective {
			if scope == configScopeUser && v.Source != config.SourceGlobal {
				continue
			}
			if scope == configScopeProject && v.Source != config.SourceProject {
				continue
			}
			if changed && isDef {
				continue
			}
		}
		spec := config.KeySpec(key)
		je := jsonEntry{
			Key:       key,
			Value:     v.Raw,
			Source:    displayConfigSource(v.Source),
			File:      v.Path,
			IsDefault: isDef,
		}
		if spec != nil {
			je.Type = string(spec.Type)
			je.IsSecret = spec.Secret
			if spec.Secret {
				je.Value = "<redacted>"
			}
		}
		entries = append(entries, je)
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetEscapeHTML(false)
	_ = showOrigin
	_ = changed
	return enc.Encode(map[string]any{
		"scope":   string(scope),
		"entries": entries,
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
			_ = flags
			eff, err := loadEffective(g)
			if err != nil {
				return err
			}
			v, err := config.Get(eff, key)
			if err != nil {
				return err
			}

			if configOutputStructured(g, cmd) {
				return writeConfigExplainJSON(cmd, key, v, eff)
			}

			return writeConfigExplainHuman(g, cmd, key, v, eff)
		},
	}
	flags.bind(cmd)
	return cmd
}

func writeConfigExplainHuman(g *globalFlags, cmd *cobra.Command, key string, v config.Value, eff *config.Effective) error {
	r := g.mustStaticRenderer(cmd)
	spec := config.KeySpec(key)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s = %s\n\n", key, diagnostics.Redact(formatConfigValue(v.Raw))))
	b.WriteString("Resolution\n")

	chain := []config.Source{config.SourceDefaults, config.SourceGlobal, config.SourceProject, config.SourceEnv, config.SourceCLI}
	for _, src := range chain {
		for k, sv := range eff.Values {
			if (k == key || config.CanonicalKey(k) == key) && sv.Source == src {
				marker := ""
				if sv.Source == v.Source {
					marker = "  <- effective"
				}
				b.WriteString(fmt.Sprintf("  %-10s %s%s\n",
					displayConfigSource(sv.Source),
					diagnostics.Redact(formatConfigValue(sv.Raw)), marker))
				break
			}
		}
	}

	b.WriteString("\nSchema\n")
	if spec != nil {
		typeStr := string(spec.Type)
		if spec.Type == config.TypeEnum {
			typeStr = "enum"
		}
		kv := []presentation.KeyValue{
			{Key: "Type", Value: typeStr},
			{Key: "Default", Value: formatConfigValue(spec.Default)},
		}
		if len(spec.Enum) > 0 {
			kv = append(kv, presentation.KeyValue{Key: "Allowed", Value: strings.Join(spec.Enum, ", ")})
		}
		if len(spec.Scopes) > 0 {
			scopes := make([]string, len(spec.Scopes))
			for i, s := range spec.Scopes {
				scopes[i] = string(s)
			}
			kv = append(kv, presentation.KeyValue{Key: "Scopes", Value: strings.Join(scopes, ", ")})
		}
		if len(spec.Commands) > 0 {
			kv = append(kv, presentation.KeyValue{Key: "Used by", Value: strings.Join(spec.Commands, ", ")})
		}
		if spec.Deprecated {
			kv = append(kv, presentation.KeyValue{Key: "Deprecated", Value: "true"})
		}
		if spec.Replacement != "" {
			kv = append(kv, presentation.KeyValue{Key: "Replaced by", Value: spec.Replacement})
		}
		b.WriteString(r.KeyValues(kv))
	}
	if legacy := config.LegacyKey(key); legacy != "" {
		b.WriteString("\n")
		b.WriteString(r.KeyValues([]presentation.KeyValue{
			{Key: "Legacy key", Value: legacy},
		}))
	}
	b.WriteString("\n")
	b.WriteString(r.KeyValues([]presentation.KeyValue{
		{Key: "Source", Value: displayConfigSource(v.Source)},
		{Key: "File", Value: v.Path, Style: presentation.ValuePath},
	}))

	return writeStaticOut(cmd, b.String())
}

func writeConfigExplainJSON(cmd *cobra.Command, key string, v config.Value, eff *config.Effective) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetEscapeHTML(false)
	spec := config.KeySpec(key)
	out := map[string]any{
		"key":             key,
		"effective_value": v.Raw,
		"source":          displayConfigSource(v.Source),
		"file":            v.Path,
	}
	if spec != nil {
		out["value"] = v.Raw
		out["type"] = string(spec.Type)
		out["default"] = spec.Default
		out["description"] = spec.Description
		if len(spec.Enum) > 0 {
			out["allowed"] = spec.Enum
		}
		scopes := make([]string, len(spec.Scopes))
		for i, s := range spec.Scopes {
			scopes[i] = string(s)
		}
		out["scopes"] = scopes
		out["is_secret"] = spec.Secret
		if spec.Secret {
			out["value"] = "<redacted>"
		}
	}
	_ = eff
	return enc.Encode(out)
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
		cli["prefer_offline"] = true
	}
	if g.configPath != "" {
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

func checkUserScopedKey(key string, target configWriteTarget) error {
	spec := config.KeySpec(key)
	if spec == nil {
		return nil
	}
	if len(spec.Scopes) == 0 {
		return nil
	}
	for _, s := range spec.Scopes {
		if string(s) == string(target.Scope) {
			return nil
		}
	}
	allowedScopes := make([]string, len(spec.Scopes))
	for i, s := range spec.Scopes {
		allowedScopes[i] = string(s)
	}
	return apperr.New(apperr.Usage, "config.set", key,
		fmt.Sprintf("%s is scoped to [%s]; cannot write to %s config",
			key, strings.Join(allowedScopes, ", "), target.Scope))
}

// readScopeValue reads a single config value from a file without going through effective merge.
func readScopeValue(filePath, key string) string {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	parsed, err := config.ParseJSONC(b)
	if err != nil {
		return ""
	}
	m, ok := parsed.(map[string]any)
	if !ok {
		return ""
	}
	flat := flattenMap(m, "")
	v, ok := flat[key]
	if !ok {
		canon := config.CanonicalKey(key)
		if canon != "" && canon != key {
			v, ok = flat[canon]
		}
	}
	if !ok {
		return ""
	}
	return formatConfigValue(v)
}

// readEffectiveValue reads the effective value for a key.
func readEffectiveValue(g *globalFlags, key string) string {
	eff, err := loadEffective(g)
	if err != nil {
		return ""
	}
	v, err := config.Get(eff, key)
	if err != nil {
		return ""
	}
	return formatConfigValue(v.Raw)
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
