package transform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// NormalizedOptions collects transform-relevant tsconfig options.
type NormalizedOptions struct {
	Target                  string              `json:"target,omitempty"`
	Module                  string              `json:"module,omitempty"`
	UseDefineForClassFields bool                `json:"useDefineForClassFields,omitempty"`
	VerbatimModuleSyntax    bool                `json:"verbatimModuleSyntax,omitempty"`
	ImportHelpers           bool                `json:"importHelpers,omitempty"`
	NoEmit                  bool                `json:"noEmit,omitempty"`
	BaseURL                 string              `json:"baseUrl,omitempty"`
	Paths                   map[string][]string `json:"paths,omitempty"`
	JSX                     string              `json:"jsx,omitempty"`
}

// NormalizedOptionsDigest returns a stable SHA-256 of the normalized options.
func (o NormalizedOptions) Digest() string {
	data, _ := json.Marshal(o)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// maxTsconfigDepth is the maximum extends chain depth.
const maxTsconfigDepth = 20

// DiscoverTsconfig searches upward from sourceDir to find the nearest tsconfig.json.
func DiscoverTsconfig(sourceDir string) (string, error) {
	dir := filepath.Clean(sourceDir)
	for {
		candidate := filepath.Join(dir, "tsconfig.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil // reached root, no tsconfig
		}
		dir = parent
	}
}

// LoadTsconfigChain loads a tsconfig and resolves its extends chain.
func LoadTsconfigChain(configPath string) ([]TsconfigFile, error) {
	return resolveExtends(configPath, nil, 0)
}

// TsconfigFile is a parsed tsconfig with metadata.
type TsconfigFile struct {
	Path   string
	Raw    map[string]any
	Digest string
	Parent *TsconfigFile
}

// resolveExtends recursively loads the extends chain.
func resolveExtends(path string, visited map[string]bool, depth int) ([]TsconfigFile, error) {
	if depth > maxTsconfigDepth {
		return nil, fmt.Errorf("tsconfig extends chain exceeds maximum depth %d (possible cycle)", maxTsconfigDepth)
	}
	if visited == nil {
		visited = make(map[string]bool)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("tsconfig %s: %w", path, err)
	}
	if visited[abs] {
		return nil, fmt.Errorf("tsconfig extends cycle detected at %s", abs)
	}
	visited[abs] = true

	raw, digest, err := parseTsconfigFile(abs)
	if err != nil {
		return nil, err
	}

	tsc := TsconfigFile{Path: abs, Raw: raw, Digest: digest}

	// resolve extends
	extends, ok := raw["extends"]
	if !ok {
		return []TsconfigFile{tsc}, nil
	}
	parentPath, ok := extends.(string)
	if !ok || parentPath == "" {
		return []TsconfigFile{tsc}, nil
	}

	// relative extends
	resolved := resolveExtendsPath(abs, parentPath)
	parents, err := resolveExtends(resolved, visited, depth+1)
	if err != nil {
		return nil, err
	}
	tsc.Parent = &parents[len(parents)-1]
	return append(parents, tsc), nil
}

// resolveExtendsPath resolves an extends path relative to the config file.
func resolveExtendsPath(configPath, extends string) string {
	if strings.HasPrefix(extends, ".") {
		base := filepath.Dir(configPath)
		resolved := filepath.Join(base, extends)
		if filepath.Ext(resolved) == "" {
			resolved += ".json"
		}
		return resolved
	}
	// Package-style extends (e.g. "@scope/tsconfig") are not resolved yet.
	// ponytail: package extends deferred to when a real use case needs it.
	return extends
}

// parseTsconfigFile reads and JSONC-parses a tsconfig.json file.
func parseTsconfigFile(path string) (map[string]any, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("reading %s: %w", path, err)
	}
	cleaned := stripJSONC(data)

	var raw map[string]any
	if err := json.Unmarshal(cleaned, &raw); err != nil {
		return nil, "", fmt.Errorf("parsing %s: %w", path, err)
	}

	// Extract compilerOptions for normalization
	co, _ := raw["compilerOptions"].(map[string]any)
	if co == nil {
		co = map[string]any{}
	}

	h := sha256.New()
	h.Write(cleaned)
	digest := hex.EncodeToString(h.Sum(nil))
	return co, digest, nil
}

// TsconfigChainDigest returns a stable digest combining all config digests in the chain.
func TsconfigChainDigest(chain []TsconfigFile) string {
	h := sha256.New()
	for _, tsc := range chain {
		h.Write([]byte(tsc.Digest))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// NormalizeOptions extracts and normalizes transform-relevant options from a chain.
func NormalizeOptions(chain []TsconfigFile) NormalizedOptions {
	opts := NormalizedOptions{}
	for _, tsc := range chain {
		applyCompilerOptions(&opts, tsc.Raw)
	}
	return opts
}

func applyCompilerOptions(opts *NormalizedOptions, raw map[string]any) {
	if v, ok := raw["target"].(string); ok && opts.Target == "" {
		opts.Target = v
	}
	if v, ok := raw["module"].(string); ok && opts.Module == "" {
		opts.Module = v
	}
	if v, ok := raw["useDefineForClassFields"].(bool); ok && !opts.UseDefineForClassFields {
		opts.UseDefineForClassFields = v
	}
	if v, ok := raw["verbatimModuleSyntax"].(bool); ok && !opts.VerbatimModuleSyntax {
		opts.VerbatimModuleSyntax = v
	}
	if v, ok := raw["importHelpers"].(bool); ok && !opts.ImportHelpers {
		opts.ImportHelpers = v
	}
	if v, ok := raw["noEmit"].(bool); ok && !opts.NoEmit {
		opts.NoEmit = v
	}
	if v, ok := raw["baseUrl"].(string); ok && opts.BaseURL == "" {
		opts.BaseURL = v
	}
	if v, ok := raw["jsx"].(string); ok && opts.JSX == "" {
		opts.JSX = v
	}
	if v, ok := raw["paths"].(map[string]any); ok {
		if opts.Paths == nil {
			opts.Paths = make(map[string][]string)
		}
		for k, pv := range v {
			if _, exists := opts.Paths[k]; !exists {
				switch pvs := pv.(type) {
				case []any:
					for _, p := range pvs {
						if ps, ok := p.(string); ok {
							opts.Paths[k] = append(opts.Paths[k], ps)
						}
					}
				case []string:
					opts.Paths[k] = append([]string(nil), pvs...)
				}
			}
		}
	}
}

// UnsupportedOptions returns tsconfig option names that are unsupported.
func UnsupportedOptions(raw map[string]any) []string {
	supported := map[string]bool{
		"target":                  true,
		"module":                  true,
		"moduleResolution":        true,
		"useDefineForClassFields": true,
		"verbatimModuleSyntax":    true,
		"importHelpers":           true,
		"noEmit":                  true,
		"baseUrl":                 true,
		"paths":                   true,
		"jsx":                     true,
		"jsxFactory":              true,
		"jsxFragmentFactory":      true,
		"jsxImportSource":         true,
		"sourceMap":               true,
		"inlineSourceMap":         true,
	}
	var unsupported []string
	for k := range raw {
		if !supported[strings.TrimSpace(k)] {
			unsupported = append(unsupported, k)
		}
	}
	sort.Strings(unsupported)
	return unsupported
}
