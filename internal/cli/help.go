package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

type helpGroup struct {
	title string
	names []string
}

type cmdHelpMeta struct {
	group      string
	examples   []string
	related    []string
	workflow   int // lower ranks first in Common workflows
	hiddenRoot bool
}

var helpGroups = []helpGroup{
	{title: "Common workflows", names: []string{"install", "add", "run", "exec", "ci", "update"}},
	{title: "Project and dependencies", names: []string{"init", "remove", "link", "dedupe", "prune", "resolve", "fetch", "lock", "patch", "publish", "pkg", "project"}},
	{title: "Run and execute", names: []string{"env", "view"}},
	{title: "Inspect and diagnose", names: []string{"ls", "outdated", "explain", "plan", "history", "snapshot", "doctor", "features", "diff", "recover", "rollback"}},
	{title: "Security and policy", names: []string{"audit", "policy", "verify", "sbom", "builds", "trust", "approve-builds"}},
	{title: "Cache, store, and artifacts", names: []string{"cache", "store", "pack", "capsule"}},
	{title: "Configuration and development", names: []string{"config", "development", "benchmark", "conformance", "version", "completion"}},
}

var commandHelpRegistry = map[string]cmdHelpMeta{
	"install":  {group: "Common workflows", workflow: 1, examples: []string{"m install", "m install --frozen-lockfile"}, related: []string{"add", "ci", "plan"}},
	"add":      {group: "Common workflows", workflow: 2, examples: []string{"m add lodash", "m add -D typescript"}},
	"run":      {group: "Common workflows", workflow: 3, examples: []string{"m run build", "m run test -- --watch"}},
	"exec":     {group: "Common workflows", workflow: 4, examples: []string{"m exec eslint ."}},
	"ci":       {group: "Common workflows", workflow: 5, examples: []string{"m ci"}},
	"update":   {group: "Common workflows", workflow: 6, examples: []string{"m update", "m update lodash"}},
	"config":   {group: "Configuration and development", examples: []string{"m config list", "m config get store.dir"}},
	"doctor":   {group: "Inspect and diagnose", examples: []string{"m doctor", "m doctor --json"}},
	"ls":       {group: "Inspect and diagnose", examples: []string{"m ls", "m ls -r"}},
	"outdated": {group: "Inspect and diagnose", examples: []string{"m outdated", "m outdated --json"}},
	"explain":  {group: "Inspect and diagnose", examples: []string{"m explain lodash"}},
	"plan":     {group: "Inspect and diagnose", examples: []string{"m plan", "m plan update"}},
	"audit":    {group: "Security and policy", examples: []string{"m audit", "m audit --fail-on high"}},
	"policy":   {group: "Security and policy", examples: []string{"m policy check"}},
	"features": {group: "Inspect and diagnose", examples: []string{"m features --format table"}},
	"project":  {group: "Project and dependencies", examples: []string{"m project info"}},
	"pkg":      {group: "Project and dependencies", examples: []string{"m pkg get name", "m pkg get version"}},
	"cache":    {group: "Cache, store, and artifacts", examples: []string{"m cache dir", "m cache verify"}},
	"store":    {group: "Cache, store, and artifacts", examples: []string{"m store status", "m store path"}},
}

func configureGroupedHelp(root *cobra.Command) {
	root.SetHelpTemplate(groupedRootHelpTemplate)
	root.SetUsageTemplate(groupedUsageTemplate)
	cobra.AddTemplateFunc("mewGroupedCommands", renderGroupedCommands)
	cobra.AddTemplateFunc("mewCommandSections", renderCommandSections)
	for _, cmd := range root.Commands() {
		applyCommandHelp(cmd)
	}
	configureTopicHelp(root)
}

func applyCommandHelp(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	if _, ok := commandHelpRegistry[cmd.Name()]; ok {
		cmd.SetHelpTemplate(commandHelpTemplate)
	}
	for _, sub := range cmd.Commands() {
		applyCommandHelp(sub)
	}
}

const groupedRootHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{- if .HasSubCommands}}Usage:
  {{.CommandPath}} [command]
{{end}}{{if .HasSubCommands}}

{{mewGroupedCommands .}}
{{- end}}

{{- if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{- end}}
{{- if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{- end}}
{{- if .HasExample}}

Examples:
{{.Example}}
{{- end}}
{{- if .HasHelpSubCommands}}

Additional help topics:
{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath 28}} {{.Short}}{{end}}{{end}}
{{- end}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
Use "{{.CommandPath}} help <topic>" for curated topics (errors, runner, lifecycle-trust, …).
`

const groupedUsageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}

{{mewGroupedCommands .}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`

const commandHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}Usage:
  {{.UseLine}}
{{mewCommandSections .}}
{{- if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{- end}}
{{- if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{- end}}
`

func renderGroupedCommands(cmd *cobra.Command) string {
	byName := map[string]*cobra.Command{}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.Hidden {
			continue
		}
		byName[c.Name()] = c
	}
	var b strings.Builder
	seen := make(map[string]struct{})
	for _, g := range helpGroups {
		var lines []string
		for _, name := range g.names {
			c, ok := byName[name]
			if !ok {
				continue
			}
			seen[name] = struct{}{}
			lines = append(lines, fmt.Sprintf("  %-14s %s", c.Name(), c.Short))
		}
		if len(lines) == 0 {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(g.title)
		b.WriteByte('\n')
		for _, line := range lines {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	var other []string
	for name, c := range byName {
		if _, ok := seen[name]; ok {
			continue
		}
		other = append(other, fmt.Sprintf("  %-14s %s", c.Name(), c.Short))
	}
	if len(other) > 0 {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("Other")
		b.WriteByte('\n')
		for _, line := range other {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderCommandSections(cmd *cobra.Command) string {
	meta, ok := commandHelpRegistry[cmd.Name()]
	if !ok {
		return ""
	}
	var b strings.Builder
	if len(meta.examples) > 0 {
		b.WriteString("\nExamples:\n")
		for _, ex := range meta.examples {
			b.WriteString("  ")
			b.WriteString(ex)
			b.WriteByte('\n')
		}
	}
	if len(meta.related) > 0 {
		b.WriteString("\nRelated:\n")
		for _, rel := range meta.related {
			b.WriteString("  ")
			b.WriteString(rel)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// writeHelp executes help for tests and golden capture.
func writeHelp(w io.Writer, root *cobra.Command, args ...string) error {
	root.SetOut(w)
	root.SetErr(w)
	root.SetArgs(args)
	return root.Execute()
}
