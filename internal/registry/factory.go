package registry

import (
	"os"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/fetch"
	"github.com/mewisme/mew/internal/project"
)

// NewFromApp builds a Client from effective config and project identity.
func NewFromApp(eff *config.Effective, projectRoot string, identity project.Identity) (*Client, error) {
	hc, err := fetch.NewClient(fetch.Options{
		Timeout:  config.Duration(eff, "network.timeout", "60s"),
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
		PreferOffline: config.Bool(eff, "prefer_offline", false),
		AuthToken:     config.AuthToken(eff),
		HTTPClient:    hc,
		MaxWorkers:    defaultMaxWorkers(),
	}), nil
}

func defaultMaxWorkers() int {
	// ponytail: cap at 16; NumCPU when unset
	w := runtime.NumCPU()
	if w > 16 {
		w = 16
	}
	if w < 1 {
		w = 1
	}
	return w
}

// ResolveBaseForPackage returns the registry URL for a package name.
func ResolveBaseForPackage(eff *config.Effective, projectRoot string, identity project.Identity, name string) string {
	return ResolveRegistryURL(name, eff, projectRoot, identity)
}
