package app

import (
	"context"

	"github.com/mewisme/mew/internal/advisory"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/registry"
)

// AuditOptions controls dependency vulnerability auditing.
type AuditOptions struct {
	Fix bool
}

// AuditResult is the outcome of m audit.
type AuditResult struct {
	Report advisory.AuditReport
	Fixes  []advisory.FixSuggestion
}

// Audit scans the project lock graph against the cached OSV advisory database.
func Audit(ctx context.Context, ac *Context, opts AuditOptions) (AuditResult, error) {
	var out AuditResult
	if ac == nil || ac.Config == nil {
		return out, apperr.New(apperr.Internal, "app.audit", "", "missing app context")
	}
	if err := ctx.Err(); err != nil {
		return out, apperr.Wrap(apperr.Cancelled, "app.audit", "", err)
	}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return out, err
	}
	g, err := LoadInstalledGraph(ctx, ac, proj)
	if err != nil {
		return out, err
	}
	store := advisory.Store{Dir: advisory.AdvisoryCacheDir(ac.Config)}
	db, err := store.Load()
	if err != nil {
		if config.Bool(ac.Config, "offline", false) && apperr.CodeOf(err) == apperr.NotFound {
			return out, apperr.New(apperr.Network, "app.audit", store.Path(),
				"advisory database not in cache (offline mode)")
		}
		return out, err
	}
	out.Report = db.MatchGraph(g)
	if !opts.Fix || len(out.Report.Vulnerabilities) == 0 {
		return out, nil
	}
	reg, err := registry.NewFromApp(ac.Config, proj.Root, proj.Identity)
	if err != nil {
		return out, err
	}
	fixes, err := advisory.SuggestFixes(ctx, db, reg, out.Report)
	if err != nil {
		return out, err
	}
	out.Fixes = fixes
	return out, nil
}
