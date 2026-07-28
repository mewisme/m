package registry

import (
	"bufio"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/project"
)

// ResolveRegistryURL picks the registry base URL for a package name.
func ResolveRegistryURL(name string, eff *config.Effective, projectRoot string, identity project.Identity) string {
	base := strings.TrimRight(config.String(eff, "registry", "https://registry.npmjs.org"), "/")
	scope := packageScope(name)
	if scope != "" {
		if u, ok := config.ScopeRegistries(eff)[scope]; ok && u != "" {
			return strings.TrimRight(u, "/")
		}
	}
	if identity != project.IdentityMew && projectRoot != "" {
		if u := readNpmrcRegistry(projectRoot, scope); u != "" {
			return strings.TrimRight(u, "/")
		}
	}
	return base
}

func packageScope(name string) string {
	if !strings.HasPrefix(name, "@") {
		return ""
	}
	i := strings.IndexByte(name, '/')
	if i <= 1 {
		return ""
	}
	return name[:i]
}

func readNpmrcRegistry(projectRoot, scope string) string {
	candidates := []string{
		filepath.Join(projectRoot, ".npmrc"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".npmrc"))
	}
	var scopeURL, defaultURL string
	for _, path := range candidates {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
				continue
			}
			key, val, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"'`)
			if scope != "" && key == scope+":registry" {
				scopeURL = val
			}
			if key == "registry" && defaultURL == "" {
				defaultURL = val
			}
		}
		_ = f.Close()
		if scopeURL != "" {
			return scopeURL
		}
	}
	if scope == "" {
		return defaultURL
	}
	return ""
}

// OriginKey returns a stable origin string for cache keys.
func OriginKey(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return strings.TrimRight(baseURL, "/")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}
