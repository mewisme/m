package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/fetch"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/store"
)

const offlinePreflightMaxLines = 20

// OfflineMissingKind classifies a missing offline prerequisite.
type OfflineMissingKind string

const (
	OfflineMissingPackument OfflineMissingKind = "packument"
	OfflineMissingBlob      OfflineMissingKind = "blob"
	OfflineMissingGit       OfflineMissingKind = "git"
	OfflineMissingLocal     OfflineMissingKind = "local"
)

// OfflineMissing is one absent cache object or local path required for offline install.
type OfflineMissing struct {
	Kind    OfflineMissingKind `json:"kind"`
	Subject string             `json:"subject"`
	Detail  string             `json:"detail,omitempty"`
	Code    apperr.Code        `json:"code"`
}

// OfflineReport summarizes offline preflight results.
type OfflineReport struct {
	Missing []OfflineMissing `json:"missing,omitempty"`
}

// OK reports whether all prerequisites are present.
func (r OfflineReport) OK() bool {
	return len(r.Missing) == 0
}

// PreflightOffline verifies registry metadata, tarball blobs, git pins, and local paths.
func PreflightOffline(ctx context.Context, ac *Context, proj *project.Project, g *graph.Graph, ext lockfile.Extensions, preExtracts map[string]string) (OfflineReport, error) {
	var report OfflineReport
	if ac == nil || ac.Config == nil || proj == nil || g == nil {
		return report, nil
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if preExtracts == nil {
		preExtracts = map[string]string{}
	}

	regClient, err := registry.NewFromApp(ac.Config, proj.Root, proj.Identity)
	if err != nil {
		return report, err
	}
	blobRoot := config.BlobCacheDir(ac.Config)
	blobStore := store.NewDir(blobRoot)

	locals, err := resolver.DecodeLocalSources(ext)
	if err != nil {
		return report, err
	}
	gits, err := resolver.DecodeGitSources(ext)
	if err != nil {
		return report, err
	}

	needPackument := map[string]string{} // name -> registry base
	for _, pkg := range g.Packages {
		key := pkg.ID.Key()
		if preExtracts[key] != "" {
			continue
		}
		if _, ok := locals[key]; ok {
			continue
		}
		if _, ok := gits[key]; ok {
			continue
		}
		if pkg.TarballURL == "" && pkg.Integrity != "" {
			base := registry.ResolveBaseForPackage(ac.Config, proj.Root, proj.Identity, pkg.ID.Name)
			needPackument[pkg.ID.Name] = base
		}
	}
	for name, base := range needPackument {
		if !packumentCached(regClient, base, name) {
			report.Missing = append(report.Missing, OfflineMissing{
				Kind:    OfflineMissingPackument,
				Subject: name,
				Detail:  "packument not in registry cache",
				Code:    apperr.Network,
			})
		}
	}

	for _, pkg := range g.Packages {
		key := pkg.ID.Key()
		if preExtracts[key] != "" {
			continue
		}
		if loc, ok := locals[key]; ok {
			if err := checkLocalSource(proj.Root, loc); err != nil {
				report.Missing = append(report.Missing, OfflineMissing{
					Kind:    OfflineMissingLocal,
					Subject: key,
					Detail:  err.Error(),
					Code:    apperr.NotFound,
				})
			}
			continue
		}
		if git, ok := gits[key]; ok {
			if strings.TrimSpace(git.Commit) == "" {
				report.Missing = append(report.Missing, OfflineMissing{
					Kind:    OfflineMissingGit,
					Subject: key,
					Detail:  "git commit not pinned in lock",
					Code:    apperr.NotFound,
				})
			} else {
				report.Missing = append(report.Missing, OfflineMissing{
					Kind:    OfflineMissingGit,
					Subject: key,
					Detail:  "git source unavailable offline",
					Code:    apperr.Network,
				})
			}
			continue
		}
		if pkg.Integrity == "" {
			continue
		}
		expected, err := fetch.ExpectedIntegrity(pkg.Integrity, "")
		if err != nil {
			report.Missing = append(report.Missing, OfflineMissing{
				Kind:    OfflineMissingBlob,
				Subject: key,
				Detail:  err.Error(),
				Code:    apperr.NotFound,
			})
			continue
		}
		exists, err := blobStore.ExistsVerified(store.Key(expected.BlobPath()))
		if err != nil {
			return report, err
		}
		if !exists {
			report.Missing = append(report.Missing, OfflineMissing{
				Kind:    OfflineMissingBlob,
				Subject: key,
				Detail:  expected.BlobPath(),
				Code:    apperr.Network,
			})
		}
	}

	sort.Slice(report.Missing, func(i, j int) bool {
		a, b := report.Missing[i], report.Missing[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Subject != b.Subject {
			return a.Subject < b.Subject
		}
		return a.Detail < b.Detail
	})
	return report, nil
}

func packumentCached(client *registry.Client, base, name string) bool {
	if client == nil {
		return false
	}
	cache := client.Cache()
	if cache == nil {
		return false
	}
	_, _, ok := cache.Lookup(registry.OriginKey(base), name)
	return ok
}

func checkLocalSource(projRoot string, loc resolver.LocalSource) error {
	switch loc.Protocol {
	case "file", "portal", "tarball", "link", "workspace":
		abs := filepath.Join(projRoot, filepath.FromSlash(loc.Path))
		if _, err := os.Stat(abs); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("local path %q not found", loc.Path)
			}
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported local protocol %q", loc.Protocol)
	}
}

func offlinePreflightError(report OfflineReport) error {
	if report.OK() {
		return nil
	}
	var b strings.Builder
	b.WriteString("offline preflight failed")
	limit := offlinePreflightMaxLines
	if len(report.Missing) < limit {
		limit = len(report.Missing)
	}
	for i := 0; i < limit; i++ {
		m := report.Missing[i]
		fmt.Fprintf(&b, "\n- %s %s: %s", m.Kind, m.Subject, m.Detail)
	}
	if len(report.Missing) > offlinePreflightMaxLines {
		fmt.Fprintf(&b, "\n... and %d more missing", len(report.Missing)-offlinePreflightMaxLines)
	}
	code := apperr.Network
	for _, m := range report.Missing {
		if m.Code == apperr.NotFound {
			code = apperr.NotFound
			break
		}
	}
	return apperr.New(code, "app.install.offline", "", b.String())
}
