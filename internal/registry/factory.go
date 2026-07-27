package registry

import (
	"os"
	"strings"
	"time"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/fetch"
	"github.com/mewisme/m/internal/project"
)

// NewFromApp builds a Client from effective config and project identity.
func NewFromApp(eff *config.Effective, projectRoot string, identity project.Identity) (*Client, error) {
	timeoutMs := config.Int(eff, "network.timeout_ms", 60000)
	hc, err := fetch.NewClient(fetch.Options{
		Timeout:  time.Duration(timeoutMs) * time.Millisecond,
		ProxyURL: config.String(eff, "network.proxy", ""),
		CAFile:   config.String(eff, "network.ca_file", ""),
	})
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(config.String(eff, "registry", "https://registry.npmjs.org"), "/")
	cacheDir := config.RegistryMetadataCacheDir(eff)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, apperr.Wrap(apperr.IO, "registry.cache", cacheDir, err)
	}
	return NewClient(Options{
		BaseURL:       base,
		CacheDir:      cacheDir,
		Offline:       config.Bool(eff, "offline", false),
		PreferOffline: config.Bool(eff, "prefer-offline", false),
		AuthToken:     config.AuthToken(eff),
		HTTPClient:    hc,
	}), nil
}

// ResolveBaseForPackage returns the registry URL for a package name.
func ResolveBaseForPackage(eff *config.Effective, projectRoot string, identity project.Identity, name string) string {
	return ResolveRegistryURL(name, eff, projectRoot, identity)
}
