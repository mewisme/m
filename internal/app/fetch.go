package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/archive"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/fetch"
	"github.com/mewisme/mew/internal/store"
)

// FetchPackage is one package entry in a fetch plan.
type FetchPackage struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	TarballURL string `json:"tarballUrl"`
	Integrity  string `json:"integrity"`
	Shasum     string `json:"shasum,omitempty"`
}

// FetchPlan is the input for m fetch --plan-file.
type FetchPlan struct {
	Packages []FetchPackage `json:"packages"`
}

// FetchResult summarizes one fetched package.
type FetchResult struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Dest      string `json:"dest"`
	BlobPath  string `json:"blobPath"`
	Integrity string `json:"integrity"`
	Size      int64  `json:"size"`
}

// CacheVerifyResult reports blob cache integrity scan results.
type CacheVerifyResult struct {
	OK   int `json:"ok"`
	Bad  int `json:"bad"`
	Skip int `json:"skip"`
}

// Fetch downloads, verifies, and extracts each package in plan under destDir.
func Fetch(ctx context.Context, ac *Context, plan FetchPlan, destDir string) ([]FetchResult, error) {
	if ac == nil || ac.Config == nil {
		return nil, apperr.New(apperr.Internal, "app.fetch", "", "missing app context")
	}
	if len(plan.Packages) == 0 {
		return nil, apperr.New(apperr.Usage, "app.fetch", "", "empty plan")
	}
	if destDir == "" {
		destDir = "."
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, apperr.Wrap(apperr.IO, "app.fetch", destDir, err)
	}
	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "app.fetch", destDir, err)
	}

	dl, err := newDownloader(ac)
	if err != nil {
		return nil, err
	}

	reqs := make([]fetch.DownloadRequest, len(plan.Packages))
	for i, p := range plan.Packages {
		reqs[i] = fetch.DownloadRequest{
			URL:       p.TarballURL,
			Integrity: p.Integrity,
			Shasum:    p.Shasum,
			AuthToken: config.AuthToken(ac.Config),
		}
	}
	arts, err := dl.DownloadAll(ctx, reqs)
	if err != nil {
		return nil, err
	}

	out := make([]FetchResult, len(plan.Packages))
	opts := archive.DefaultOptions()
	for i, p := range plan.Packages {
		pkgDest := filepath.Join(destAbs, fmt.Sprintf("%s@%s", p.Name, p.Version))
		if err := os.RemoveAll(pkgDest); err != nil {
			return nil, apperr.Wrap(apperr.IO, "app.fetch", pkgDest, err)
		}
		if err := archive.Extract(ctx, arts[i].BlobPath, pkgDest, opts); err != nil {
			_ = os.RemoveAll(pkgDest) // intentional: best-effort cleanup of a partial extract; the extract error below is authoritative
			return nil, err
		}
		out[i] = FetchResult{
			Name:      p.Name,
			Version:   p.Version,
			Dest:      pkgDest,
			BlobPath:  arts[i].BlobPath,
			Integrity: arts[i].Integrity.Original,
			Size:      arts[i].Size,
		}
	}
	return out, nil
}

// VerifyBlobCache re-hashes every blob under the cache and reports mismatches.
func VerifyBlobCache(ctx context.Context, ac *Context) (CacheVerifyResult, error) {
	var res CacheVerifyResult
	if ac == nil || ac.Config == nil {
		return res, apperr.New(apperr.Internal, "app.cache.verify", "", "missing app context")
	}
	root := config.BlobCacheDir(ac.Config)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return res, nil
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) != 2 {
			res.Skip++
			return nil
		}
		algo, hex := parts[0], parts[1]
		f, err := os.Open(path)
		if err != nil {
			return apperr.Wrap(apperr.IO, "app.cache.verify", path, err)
		}
		parsed, _, verr := fetch.VerifyReader(f, algo+"-"+hex, "")
		_ = f.Close() // intentional: read-only handle; close error cannot affect the verification result
		if verr != nil || parsed.Hex != strings.ToLower(hex) {
			res.Bad++
			return nil
		}
		res.OK++
		return nil
	})
	return res, err
}

func newDownloader(ac *Context) (*fetch.Downloader, error) {
	hc, err := fetch.NewClient(fetch.Options{
		Timeout:  config.Duration(ac.Config, "network.timeout", "60s"),
		ProxyURL: config.String(ac.Config, "network.proxy", ""),
		CAFile:   config.String(ac.Config, "network.ca_file", ""),
	})
	if err != nil {
		return nil, err
	}
	blobRoot := config.BlobCacheDir(ac.Config)
	if err := os.MkdirAll(blobRoot, 0o755); err != nil {
		return nil, apperr.Wrap(apperr.IO, "app.fetch", blobRoot, err)
	}
	staging := filepath.Join(config.CacheRoot(ac.Config), "staging")
	return &fetch.Downloader{
		Client:        hc,
		Store:         store.NewDir(blobRoot),
		StagingDir:    staging,
		Workers:       fetch.DefaultWorkers(),
		Offline:       config.Bool(ac.Config, "offline", false),
		PreferOffline: config.Bool(ac.Config, "prefer_offline", false),
	}, nil
}

// LoadFetchPlan reads a JSON fetch plan from path.
func LoadFetchPlan(path string) (FetchPlan, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return FetchPlan{}, apperr.Wrap(apperr.IO, "app.fetch.plan", path, err)
	}
	var plan FetchPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return FetchPlan{}, apperr.Wrap(apperr.Usage, "app.fetch.plan", path, err)
	}
	return plan, nil
}
