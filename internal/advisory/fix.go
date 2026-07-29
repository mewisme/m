package advisory

import (
	"context"
	"path/filepath"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/semver"
)

// AdvisoryCacheDir is <cache>/advisory.
func AdvisoryCacheDir(eff *config.Effective) string {
	return filepath.Join(config.CacheRoot(eff), "advisory")
}

// SuggestFixes returns sorted safe version bumps for report findings.
func SuggestFixes(ctx context.Context, db *AdvisoryDB, reg *registry.Client, report AuditReport) ([]FixSuggestion, error) {
	if db == nil || reg == nil || len(report.Vulnerabilities) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, apperr.Wrap(apperr.Cancelled, "advisory.fix", "", err)
	}
	byPkg := map[string]Vulnerability{}
	for _, v := range report.Vulnerabilities {
		if prev, ok := byPkg[v.Package]; !ok || v.Version < prev.Version {
			byPkg[v.Package] = v
		}
	}
	var out []FixSuggestion
	for name, vuln := range byPkg {
		p, err := reg.Packument(ctx, reg.BaseURL(), name)
		if err != nil {
			return nil, err
		}
		versions := p.SortedVersionsSemver()
		sort.Slice(versions, func(i, j int) bool {
			c, _ := semver.Compare(versions[i], versions[j])
			return c < 0
		})
		var fix string
		for _, ver := range versions {
			cmp, err := semver.Compare(ver, vuln.Version)
			if err != nil || cmp <= 0 {
				continue
			}
			if !db.IsVulnerable(name, ver) {
				fix = ver
				break
			}
		}
		if fix == "" {
			continue
		}
		out = append(out, FixSuggestion{
			Package:       name,
			FromVersion:   vuln.Version,
			ToVersion:     fix,
			Vulnerability: vuln.ID,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Package != out[j].Package {
			return out[i].Package < out[j].Package
		}
		return out[i].FromVersion < out[j].FromVersion
	})
	return out, nil
}
