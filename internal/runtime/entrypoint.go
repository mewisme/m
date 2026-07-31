package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// supportedExts is the set of extensions we recognize as JS entrypoints.
var supportedExts = map[string]bool{
	".js":  true,
	".mjs": true,
	".cjs": true,
}

// IsJSFile reports whether the selector looks like a JS file (has a supported extension
// or contains a directory separator).
func IsJSFile(selector string) bool {
	if selector == "" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(selector))
	if supportedExts[ext] {
		return true
	}
	// A path with a directory separator is likely a file reference.
	if strings.ContainsAny(selector, "/\\") {
		return true
	}
	return false
}

// ResolveEntrypoint resolves a selector to an absolute filesystem path.
// Returns the absolute path, or a typed error if the entrypoint is invalid.
func ResolveEntrypoint(cwd, selector string) (string, error) {
	if selector == "" {
		return "", apperr.New(apperr.RuntimeEntrypoint, "runtime.entrypoint", "", "empty entrypoint")
	}
	if !filepath.IsAbs(selector) {
		selector = filepath.Join(cwd, selector)
	}
	abs, err := filepath.Abs(selector)
	if err != nil {
		return "", apperr.Wrap(apperr.RuntimeEntrypoint, "runtime.entrypoint", selector, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", apperr.New(apperr.RuntimeEntrypoint, "runtime.entrypoint", abs,
				fmt.Sprintf("entrypoint not found: %s", abs))
		}
		return "", apperr.Wrap(apperr.RuntimeEntrypoint, "runtime.entrypoint", abs, err)
	}

	if info.IsDir() {
		return "", apperr.New(apperr.RuntimeEntrypoint, "runtime.entrypoint", abs,
			"entrypoint is a directory, not a file")
	}

	ext := strings.ToLower(filepath.Ext(abs))
	if !supportedExts[ext] {
		return "", apperr.New(apperr.RuntimeEntrypoint, "runtime.entrypoint", abs,
			fmt.Sprintf("unsupported file extension %q; expected .js, .mjs, or .cjs", ext))
	}

	return abs, nil
}
