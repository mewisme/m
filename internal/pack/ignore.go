package pack

import (
	"path/filepath"
	"strings"
)

// ponytail: ignore matching is line-order glob only; upgrade = npm ignore-walk parity.

var defaultIgnoreLines = []string{
	".git",
	"node_modules",
	".mew",
	".DS_Store",
	"*.tgz",
	"*.tmp",
	".npm",
}

func parseIgnoreLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func loadIgnorePatterns(root string) []string {
	patterns := append([]string(nil), defaultIgnoreLines...)
	if data, err := readFileOptional(filepath.Join(root, ".npmignore")); err == nil && len(data) > 0 {
		patterns = append(patterns, parseIgnoreLines(data)...)
		return patterns
	}
	if data, err := readFileOptional(filepath.Join(root, ".gitignore")); err == nil && len(data) > 0 {
		patterns = append(patterns, parseIgnoreLines(data)...)
	}
	return patterns
}

func ignoredPath(rel string, patterns []string) bool {
	rel = filepath.ToSlash(rel)
	ignored := false
	for _, pat := range patterns {
		neg := false
		if strings.HasPrefix(pat, "!") {
			neg = true
			pat = strings.TrimSpace(pat[1:])
		}
		if pat == "" {
			continue
		}
		if matchIgnorePattern(rel, pat) {
			if neg {
				ignored = false
			} else {
				ignored = true
			}
		}
	}
	return ignored
}

func matchIgnorePattern(rel, pat string) bool {
	pat = filepath.ToSlash(pat)
	if strings.HasSuffix(pat, "/") {
		pat = strings.TrimSuffix(pat, "/")
		if rel == pat || strings.HasPrefix(rel, pat+"/") {
			return true
		}
	}
	if strings.Contains(pat, "*") || strings.Contains(pat, "?") {
		if ok, _ := filepath.Match(pat, rel); ok {
			return true
		}
		base := filepath.Base(rel)
		if ok, _ := filepath.Match(pat, base); ok {
			return true
		}
		return false
	}
	return rel == pat || strings.HasPrefix(rel, pat+"/")
}
