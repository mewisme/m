// Package help provides the embedded terminal-help topic registry.
//
// Domain-safe: stdlib + embed only. No Charm / presentation imports.
package help

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	terminalhelp "github.com/mewisme/mew/docs/terminal-help"
	"github.com/mewisme/mew/internal/apperr"
)

// Topic is one curated terminal-help document.
type Topic struct {
	ID           string
	Title        string
	Summary      string
	SourcePath   string
	Aliases      []string
	Related      []string
	Experimental bool
}

// Registry is an immutable validated topic catalog backed by embedded Markdown.
type Registry struct {
	topics  []Topic
	byID    map[string]*Topic
	byAlias map[string]*Topic
	files   map[string]struct{}
	content fs.FS
}

// Default returns the built-in embedded registry.
func Default() (*Registry, error) {
	return New(terminalhelp.FS, catalog())
}

// New validates catalog metadata against an embed FS and returns a registry.
func New(content fs.FS, topics []Topic) (*Registry, error) {
	if content == nil {
		return nil, apperr.New(apperr.Internal, "help.registry", "", "nil content FS")
	}
	files, err := listMarkdown(content)
	if err != nil {
		return nil, err
	}
	r := &Registry{
		topics:  append([]Topic(nil), topics...),
		byID:    make(map[string]*Topic, len(topics)),
		byAlias: make(map[string]*Topic),
		files:   files,
		content: content,
	}
	sort.Slice(r.topics, func(i, j int) bool { return r.topics[i].ID < r.topics[j].ID })
	if err := r.validate(); err != nil {
		return nil, err
	}
	return r, nil
}

func catalog() []Topic {
	return []Topic{
		{
			ID:         "errors",
			Title:      "Error help index",
			Summary:    "Index of curated ERR_M_* terminal topics",
			SourcePath: "docs/terminal-help/errors.md",
			Related:    []string{"errors/ERR_M_LOCKFILE", "errors/ERR_M_POLICY", "errors/ERR_M_INTEGRITY", "errors/ERR_M_TRANSACTION", "errors/ERR_M_USAGE", "errors/ERR_M_CANCELLED"},
		},
		{
			ID:         "errors/ERR_M_LOCKFILE",
			Title:      "ERR_M_LOCKFILE",
			Summary:    "Lockfile parse, checksum, graph, or frozen drift",
			SourcePath: "docs/terminal-help/errors/ERR_M_LOCKFILE.md",
			Aliases:    []string{"ERR_M_LOCKFILE"},
			Related:    []string{"errors", "errors/ERR_M_TRANSACTION"},
		},
		{
			ID:         "errors/ERR_M_POLICY",
			Title:      "ERR_M_POLICY",
			Summary:    "Lifecycle trust or org policy block",
			SourcePath: "docs/terminal-help/errors/ERR_M_POLICY.md",
			Aliases:    []string{"ERR_M_POLICY"},
			Related:    []string{"errors", "lifecycle-trust"},
		},
		{
			ID:         "errors/ERR_M_INTEGRITY",
			Title:      "ERR_M_INTEGRITY",
			Summary:    "Checksum, provenance, or ambiguous recovery state",
			SourcePath: "docs/terminal-help/errors/ERR_M_INTEGRITY.md",
			Aliases:    []string{"ERR_M_INTEGRITY"},
			Related:    []string{"errors", "errors/ERR_M_TRANSACTION"},
		},
		{
			ID:         "errors/ERR_M_TRANSACTION",
			Title:      "ERR_M_TRANSACTION",
			Summary:    "Transaction journal, commit, rollback, or lock failure",
			SourcePath: "docs/terminal-help/errors/ERR_M_TRANSACTION.md",
			Aliases:    []string{"ERR_M_TRANSACTION"},
			Related:    []string{"errors", "snapshots"},
		},
		{
			ID:         "errors/ERR_M_USAGE",
			Title:      "ERR_M_USAGE",
			Summary:    "Invalid arguments or flag misuse",
			SourcePath: "docs/terminal-help/errors/ERR_M_USAGE.md",
			Aliases:    []string{"ERR_M_USAGE"},
			Related:    []string{"errors", "configuration", "runner"},
		},
		{
			ID:         "errors/ERR_M_CANCELLED",
			Title:      "ERR_M_CANCELLED",
			Summary:    "Interrupt or context cancellation",
			SourcePath: "docs/terminal-help/errors/ERR_M_CANCELLED.md",
			Aliases:    []string{"ERR_M_CANCELLED"},
			Related:    []string{"errors", "errors/ERR_M_TRANSACTION"},
		},
		{
			ID:         "compatibility",
			Title:      "Compatibility",
			Summary:    "Independent compatibility axes summary",
			SourcePath: "docs/terminal-help/compatibility.md",
			Aliases:    []string{"compat"},
			Related:    []string{"configuration", "runner"},
		},
		{
			ID:         "lifecycle-trust",
			Title:      "Lifecycle trust",
			Summary:    "Lifecycle script trust modes and ask prompts",
			SourcePath: "docs/terminal-help/lifecycle-trust.md",
			Aliases:    []string{"lifecycle", "trust"},
			Related:    []string{"errors/ERR_M_POLICY", "configuration"},
		},
		{
			ID:         "snapshots",
			Title:      "Snapshots",
			Summary:    "Install snapshot inspection and restore safety",
			SourcePath: "docs/terminal-help/snapshots.md",
			Aliases:    []string{"snapshot"},
			Related:    []string{"capsules", "errors/ERR_M_TRANSACTION"},
		},
		{
			ID:         "capsules",
			Title:      "Capsules",
			Summary:    "Portable dependency capsule safety notes",
			SourcePath: "docs/terminal-help/capsules.md",
			Aliases:    []string{"capsule"},
			Related:    []string{"snapshots", "errors/ERR_M_INTEGRITY"},
		},
		{
			ID:         "runner",
			Title:      "Runner",
			Summary:    "m run, m exec, and mx execution surfaces",
			SourcePath: "docs/terminal-help/runner.md",
			Related:    []string{"configuration", "lifecycle-trust"},
		},
		{
			ID:         "configuration",
			Title:      "Configuration",
			Summary:    "Layered config and UI-related keys",
			SourcePath: "docs/terminal-help/configuration.md",
			Aliases:    []string{"config"},
			Related:    []string{"runner", "lifecycle-trust"},
		},
	}
}

func (r *Registry) validate() error {
	claimed := make(map[string]string, len(r.topics))
	for i := range r.topics {
		t := &r.topics[i]
		if t.ID == "" {
			return apperr.New(apperr.Internal, "help.registry", "", "empty topic id")
		}
		if t.Title == "" || t.Summary == "" || t.SourcePath == "" {
			return apperr.New(apperr.Internal, "help.registry", t.ID, "topic missing title, summary, or source path")
		}
		if !strings.HasPrefix(t.SourcePath, "docs/terminal-help/") {
			return apperr.New(apperr.Internal, "help.registry", t.ID, "source path must be under docs/terminal-help/")
		}
		rel := strings.TrimPrefix(t.SourcePath, "docs/terminal-help/")
		if _, ok := r.files[rel]; !ok {
			return apperr.New(apperr.Internal, "help.registry", t.ID, fmt.Sprintf("missing embed file %q", rel))
		}
		if _, ok := r.byID[t.ID]; ok {
			return apperr.New(apperr.Internal, "help.registry", t.ID, "duplicate topic id")
		}
		r.byID[t.ID] = t
		claimed[rel] = t.ID
		for _, a := range t.Aliases {
			a = strings.TrimSpace(a)
			if a == "" {
				continue
			}
			if _, ok := r.byID[a]; ok {
				return apperr.New(apperr.Internal, "help.registry", t.ID, fmt.Sprintf("alias %q collides with topic id", a))
			}
			if prev, ok := r.byAlias[a]; ok {
				return apperr.New(apperr.Internal, "help.registry", t.ID, fmt.Sprintf("duplicate alias %q (also %s)", a, prev.ID))
			}
			r.byAlias[a] = t
		}
	}
	for i := range r.topics {
		t := &r.topics[i]
		for _, relID := range t.Related {
			if _, ok := r.byID[relID]; !ok {
				if _, ok := r.byAlias[relID]; !ok {
					return apperr.New(apperr.Internal, "help.registry", t.ID, fmt.Sprintf("related topic %q not found", relID))
				}
			}
		}
	}
	for path := range r.files {
		if _, ok := claimed[path]; !ok {
			return apperr.New(apperr.Internal, "help.registry", path, "orphan embed file")
		}
	}
	return nil
}

func listMarkdown(content fs.FS) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	err := fs.WalkDir(content, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		out[filepathSlash(path)] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "help.registry", "", err)
	}
	return out, nil
}

func filepathSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// Topics returns topics in deterministic ID order.
func (r *Registry) Topics() []Topic {
	if r == nil {
		return nil
	}
	out := make([]Topic, len(r.topics))
	copy(out, r.topics)
	return out
}

// Lookup resolves a topic id or alias and returns topic metadata plus Markdown bytes.
func (r *Registry) Lookup(id string) (*Topic, []byte, error) {
	if r == nil {
		return nil, nil, apperr.New(apperr.Internal, "help.lookup", id, "nil registry")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, nil, apperr.New(apperr.Usage, "help.lookup", "", "empty topic id")
	}
	t := r.byID[id]
	if t == nil {
		t = r.byAlias[id]
	}
	if t == nil {
		return nil, nil, apperr.New(apperr.Usage, "help.lookup", id, fmt.Sprintf("unknown help topic %q", id))
	}
	rel := strings.TrimPrefix(t.SourcePath, "docs/terminal-help/")
	b, err := fs.ReadFile(r.content, rel)
	if err != nil {
		return nil, nil, apperr.Wrap(apperr.Internal, "help.lookup", t.ID, err)
	}
	cp := *t
	return &cp, b, nil
}

// LookupError resolves m help errors [CODE].
func (r *Registry) LookupError(code string) (*Topic, []byte, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return r.Lookup("errors")
	}
	if strings.HasPrefix(code, "errors/") {
		return r.Lookup(code)
	}
	if strings.HasPrefix(code, "ERR_M_") {
		return r.Lookup("errors/" + code)
	}
	return r.Lookup(code)
}

// ResolveArgs maps help argv after `m help` to a topic.
//
//	[] -> nil (caller shows root/command help)
//	["errors"] / ["errors", "ERR_M_*"] -> error topics
//	[id] -> topic lookup
func (r *Registry) ResolveArgs(args []string) (*Topic, []byte, error) {
	if len(args) == 0 {
		return nil, nil, nil
	}
	if args[0] == "errors" {
		code := ""
		if len(args) > 1 {
			code = args[1]
		}
		if len(args) > 2 {
			return nil, nil, apperr.New(apperr.Usage, "help.resolve", strings.Join(args, " "),
				"usage: m help errors [ERR_M_CODE]")
		}
		return r.LookupError(code)
	}
	if len(args) != 1 {
		return nil, nil, apperr.New(apperr.Usage, "help.resolve", strings.Join(args, " "),
			"usage: m help <topic>")
	}
	return r.Lookup(args[0])
}

// FormatTopicList returns a stable multi-line topic listing for usage errors.
func (r *Registry) FormatTopicList() string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("Available topics:\n")
	for _, t := range r.topics {
		if strings.HasPrefix(t.ID, "errors/") {
			continue
		}
		b.WriteString("  ")
		b.WriteString(t.ID)
		b.WriteString("  ")
		b.WriteString(t.Summary)
		b.WriteByte('\n')
	}
	b.WriteString("  errors [ERR_M_CODE]  Error help index or code topic\n")
	return strings.TrimRight(b.String(), "\n")
}
