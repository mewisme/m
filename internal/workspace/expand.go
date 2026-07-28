// Package workspace expands package.json workspace globs into member paths.
package workspace

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
	"github.com/mewisme/mew/internal/manifest"
)

// Index is a read-only workspace membership snapshot.
type Index struct {
	Root     string
	Patterns []string
	Members  []string // root-relative slash paths, sorted
}

// Expand expands workspace patterns under root into sorted unique member dirs
// that contain package.json. Paths use forward slashes relative to root.
func Expand(root string, patterns []string) ([]string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "workspace.expand", root, err)
	}
	var includes, excludes []string
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "!") {
			excludes = append(excludes, expandBraces(p[1:])...)
		} else {
			includes = append(includes, expandBraces(p)...)
		}
	}
	seen := map[string]struct{}{}
	for _, pat := range includes {
		matched, err := matchGlob(abs, pat)
		if err != nil {
			return nil, err
		}
		for _, m := range matched {
			rel, err := filepath.Rel(abs, m)
			if err != nil {
				return nil, apperr.Wrap(apperr.IO, "workspace.expand", m, err)
			}
			rel = filepath.ToSlash(rel)
			if rel == "." || strings.HasPrefix(rel, "../") {
				continue
			}
			if _, err := os.Stat(filepath.Join(m, "package.json")); err != nil {
				continue
			}
			if err := fsx.GuardAncestors(abs, m); err != nil {
				return nil, apperr.Wrap(apperr.IO, "workspace.expand", rel, err)
			}
			if _, ok := seen[rel]; ok {
				continue
			}
			seen[rel] = struct{}{}
		}
	}
	for _, pat := range excludes {
		matched, err := matchGlob(abs, pat)
		if err != nil {
			return nil, err
		}
		for _, m := range matched {
			rel, err := filepath.Rel(abs, m)
			if err != nil {
				continue
			}
			rel = filepath.ToSlash(rel)
			delete(seen, rel)
		}
	}
	out := make([]string, 0, len(seen))
	for m := range seen {
		out = append(out, m)
	}
	sort.Strings(out)
	return out, nil
}

// BuildIndex loads root package.json workspaces and expands members.
func BuildIndex(root string) (*Index, error) {
	doc, err := manifest.LoadCached(root)
	if err != nil {
		return nil, err
	}
	patterns, err := doc.WorkspacePatterns()
	if err != nil {
		return nil, err
	}
	members, err := Expand(root, patterns)
	if err != nil {
		return nil, err
	}
	if err := CheckCycles(root, members); err != nil {
		return nil, err
	}
	abs, _ := filepath.Abs(root)
	return &Index{Root: abs, Patterns: patterns, Members: members}, nil
}

// CheckCycles rejects members whose workspaces escape the project or re-include an ancestor.
func CheckCycles(root string, members []string) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return apperr.Wrap(apperr.IO, "workspace.cycle", root, err)
	}
	for _, mem := range members {
		memAbs := filepath.Join(abs, filepath.FromSlash(mem))
		doc, err := manifest.Load(filepath.Join(memAbs, "package.json"))
		if err != nil {
			if apperr.CodeOf(err) == apperr.NotFound {
				continue
			}
			return err
		}
		patterns, err := doc.WorkspacePatterns()
		if err != nil {
			return err
		}
		for _, pat := range patterns {
			pat = strings.TrimSpace(pat)
			if pat == "" || strings.HasPrefix(pat, "!") {
				continue
			}
			for _, p := range expandBraces(pat) {
				static := staticGlobPrefix(p)
				target := filepath.Clean(filepath.Join(memAbs, filepath.FromSlash(static)))
				relRoot, err := filepath.Rel(abs, target)
				if err != nil || relRoot == ".." || strings.HasPrefix(relRoot, ".."+string(filepath.Separator)) {
					return apperr.New(apperr.Manifest, "workspace.cycle", mem,
						"cyclic workspace definition: member workspaces escape the project root")
				}
				if isAncestorOrEqual(target, memAbs) && target != memAbs {
					return apperr.New(apperr.Manifest, "workspace.cycle", mem,
						"cyclic workspace definition: member workspaces include an ancestor")
				}
				if target == abs {
					return apperr.New(apperr.Manifest, "workspace.cycle", mem,
						"cyclic workspace definition: member workspaces include the project root")
				}
			}
		}
	}
	return nil
}

func staticGlobPrefix(pat string) string {
	parts := strings.Split(filepath.ToSlash(pat), "/")
	var out []string
	for _, p := range parts {
		if strings.ContainsAny(p, "*?[") {
			break
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return "."
	}
	return strings.Join(out, "/")
}

func isAncestorOrEqual(ancestor, child string) bool {
	ancestor = filepath.Clean(ancestor)
	child = filepath.Clean(child)
	if ancestor == child {
		return true
	}
	rel, err := filepath.Rel(ancestor, child)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func expandBraces(pat string) []string {
	start := strings.IndexByte(pat, '{')
	if start < 0 {
		return []string{pat}
	}
	end := strings.IndexByte(pat[start:], '}')
	if end < 0 {
		return []string{pat}
	}
	end += start
	inner := pat[start+1 : end]
	prefix := pat[:start]
	suffix := pat[end+1:]
	parts := strings.Split(inner, ",")
	var out []string
	for _, part := range parts {
		out = append(out, expandBraces(prefix+part+suffix)...)
	}
	return out
}

func matchGlob(root, pattern string) ([]string, error) {
	pattern = path.Clean("/" + filepath.ToSlash(pattern))
	pattern = strings.TrimPrefix(pattern, "/")
	parts := strings.Split(pattern, "/")
	return matchParts(root, parts)
}

func matchParts(dir string, parts []string) ([]string, error) {
	if len(parts) == 0 {
		return []string{dir}, nil
	}
	part := parts[0]
	rest := parts[1:]
	if part == "." || part == "" {
		return matchParts(dir, rest)
	}
	if !strings.ContainsAny(part, "*?[") {
		next := filepath.Join(dir, part)
		st, err := os.Stat(next)
		if err != nil || !st.IsDir() {
			return nil, nil
		}
		return matchParts(next, rest)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "workspace.expand", dir, err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ok, err := filepath.Match(part, e.Name())
		if err != nil {
			return nil, apperr.Wrap(apperr.Manifest, "workspace.expand", part, err)
		}
		if !ok {
			continue
		}
		matched, err := matchParts(filepath.Join(dir, e.Name()), rest)
		if err != nil {
			return nil, err
		}
		out = append(out, matched...)
	}
	return out, nil
}

// PatternsFromRaw is a convenience for tests.
func PatternsFromRaw(raw json.RawMessage) ([]string, error) {
	return manifest.ParseWorkspacesField(raw)
}
