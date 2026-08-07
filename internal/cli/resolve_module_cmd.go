package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/transform"
)

func newResolveModuleCmd() *cobra.Command {
	var fromDir string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "resolve-module <specifier>",
		Short: "Show how Mew resolves a module specifier",
		Long: `Show the tsconfig paths, baseUrl, and project layout that Mew
uses to resolve a module specifier at runtime.

This is a diagnostic tool. It prints the matching path aliases and
the resolved candidate files without running Node.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "resolve-module", "", "no app context")
			}

			specifier := args[0]
			cwd := ac.CWD
			if fromDir != "" {
				cwd = fromDir
			}
			if !filepath.IsAbs(cwd) {
				var err error
				cwd, err = filepath.Abs(cwd)
				if err != nil {
					return apperr.Wrap(apperr.IO, "resolve-module", cwd, err)
				}
			}

			// Discover tsconfig from the target directory.
			configPath, err := transform.DiscoverTsconfig(cwd)
			if err != nil {
				return apperr.Wrap(apperr.TransformConfigParse, "resolve-module", cwd, err)
			}

			if jsonOutput {
				return renderResolveModuleJSON(cmd, specifier, cwd, configPath)
			}
			return renderResolveModuleText(cmd, specifier, cwd, configPath)
		},
	}

	cmd.Flags().StringVar(&fromDir, "from", "", "directory to resolve from (default: cwd)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")

	return cmd
}

func renderResolveModuleText(cmd *cobra.Command, specifier, cwd, configPath string) error {
	w := cmd.OutOrStdout()

	fmt.Fprintf(w, "Specifier: %s\n", specifier)
	fmt.Fprintf(w, "From:      %s\n", cwd)

	if configPath == "" {
		fmt.Fprintln(w, "Tsconfig:  (none found)")
		fmt.Fprintln(w, "\nNo tsconfig paths configured. Resolution falls through to Node defaults.")
		return nil
	}

	fmt.Fprintf(w, "Tsconfig:  %s\n", configPath)

	chain, err := transform.LoadTsconfigChain(configPath)
	if err != nil {
		return apperr.Wrap(apperr.TransformConfigParse, "resolve-module", configPath, err)
	}

	opts, err := transform.NormalizeOptions(chain)
	if err != nil {
		return apperr.Wrap(apperr.TransformConfigOption, "resolve-module", configPath, err)
	}

	// Show baseUrl
	baseDir := filepath.Dir(configPath)
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "."
	}
	fmt.Fprintf(w, "BaseUrl:   %s", baseURL)
	if !filepath.IsAbs(baseURL) {
		resolved := filepath.Join(baseDir, baseURL)
		fmt.Fprintf(w, "  (resolved: %s)", resolved)
	}
	fmt.Fprintln(w)

	// Show paths
	if len(opts.Paths) == 0 {
		fmt.Fprintln(w, "Paths:     (none)")
		fmt.Fprintln(w, "\nNo path aliases configured. Resolution falls through to Node defaults.")
		return nil
	}

	fmt.Fprintln(w, "Paths:")
	for _, pm := range opts.PathMappings {
		fmt.Fprintf(w, "  %s → %s\n", pm.Pattern, strings.Join(pm.Targets, ", "))
	}

	// Match the specifier against paths (in canonical specificity order).
	fmt.Fprintf(w, "\nMatching %q against path patterns:\n", specifier)
	matched := false
	for _, pm := range opts.PathMappings {
		captures := matchPathPattern(specifier, pm.Pattern)
		if captures == nil {
			continue
		}
		matched = true
		fmt.Fprintf(w, "  %s matched (captures: %v)\n", pm.Pattern, captures)
		resolveBase := baseDir
		if baseURL != "." && baseURL != "" {
			resolveBase = filepath.Join(baseDir, baseURL)
		}
		for _, replacement := range pm.Targets {
			resolved := replacement
			for _, cap := range captures {
				resolved = strings.Replace(resolved, "*", cap, 1)
			}
			full := filepath.Join(resolveBase, resolved)
			fmt.Fprintf(w, "    → %s\n", full)
		}
	}
	if !matched {
		fmt.Fprintln(w, "  (no patterns matched)")
	}

	// Check PnP
	pnpRoot := findPnpRoot(cwd)
	if pnpRoot != "" {
		fmt.Fprintf(w, "\nPnP root:  %s\n", pnpRoot)
	}

	return nil
}

func renderResolveModuleJSON(cmd *cobra.Command, specifier, cwd, configPath string) error {
	// ponytail: JSON output deferred to 0054; text output covers MVP needs.
	return renderResolveModuleText(cmd, specifier, cwd, configPath)
}

// matchPathPattern is a Go port of the ts-loader.mjs path-matching logic.
func matchPathPattern(specifier, pattern string) []string {
	if !strings.Contains(pattern, "*") {
		if specifier == pattern {
			return []string{""}
		}
		return nil
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		prefix, suffix := parts[0], parts[1]
		sufLen := len(suffix)
		if strings.HasPrefix(specifier, prefix) && strings.HasSuffix(specifier, suffix) &&
			len(specifier) >= len(prefix)+sufLen {
			captured := specifier[len(prefix) : len(specifier)-sufLen]
			return []string{captured}
		}
		return nil
	}
	// Multiple wildcards: sequential match.
	remaining := specifier
	var captures []string
	for i, part := range parts {
		switch {
		case i == 0:
			if !strings.HasPrefix(remaining, part) {
				return nil
			}
			remaining = remaining[len(part):]
		case i == len(parts)-1:
			if part == "" {
				captures = append(captures, remaining)
			} else if !strings.HasSuffix(remaining, part) {
				return nil
			} else {
				captures = append(captures, remaining[:len(remaining)-len(part)])
			}
		default:
			idx := strings.Index(remaining, part)
			if idx == -1 {
				return nil
			}
			captures = append(captures, remaining[:idx])
			remaining = remaining[idx+len(part):]
		}
	}
	return captures
}

// findPnpRoot walks up from dir looking for .pnp.cjs or .pnp.data.json.
func findPnpRoot(dir string) string {
	current := dir
	for {
		candidate := filepath.Join(current, ".pnp.cjs")
		if _, err := os.Stat(candidate); err == nil {
			return current
		}
		candidate = filepath.Join(current, ".pnp.data.json")
		if _, err := os.Stat(candidate); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}
