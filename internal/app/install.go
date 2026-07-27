package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/plan"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/transaction"
)

// InstallOptions controls m install / ci / update.
type InstallOptions struct {
	Prod          bool
	Frozen        bool
	DryRun        bool
	KeepJournal   bool
	Linker        string // hoisted | isolated | empty
	WriteManifest bool   // commit package.json when true
	Update        *UpdateResolveOptions
	// PreResolvedGraph skips resolve and uses the given frozen graph (snapshot restore).
	PreResolvedGraph *graph.Graph
	// StagedManifest writes these bytes to staged package.json (snapshot restore).
	StagedManifest []byte
	// StagedLock writes these bytes to staged m.lock (snapshot restore).
	StagedLock []byte
	// SkipSnapshot omits staging a new install snapshot in the txn (snapshot restore).
	SkipSnapshot bool
	// AddSpec is set by m add; packument fetch runs under mutation ownership via prepareAddDependency.
	AddSpec      string
	AddDev       bool
	AddSaveExact bool
}

// UpdateResolveOptions selects incremental update resolve parameters.
type UpdateResolveOptions struct {
	Targets           []string
	PriorOverrides    map[string]string
	PriorFingerprints *resolver.PriorFingerprints
}

// InstallResult summarizes package changes.
type InstallResult struct {
	Added               int        `json:"added"`
	Removed             int        `json:"removed"`
	Changed             int        `json:"changed"`
	Packages            int        `json:"packages"`
	Plan                *plan.Plan `json:"plan,omitempty"`
	Committed           bool       `json:"committed,omitempty"`
	RolledBack          bool       `json:"rolledBack,omitempty"`
	RecoveryRequired    bool       `json:"recoveryRequired,omitempty"`
	CleanupIncomplete   bool       `json:"cleanupIncomplete,omitempty"`
	CleanupWarningCodes []string   `json:"cleanupWarningCodes,omitempty"`
	CleanupWarnings     []string   `json:"cleanupWarnings,omitempty"`
}

// AddOptions controls m add.
type AddOptions struct {
	Dev       bool
	SaveExact bool
	Install   InstallOptions
}

// Install resolves, fetches, links, and commits via journaled transaction.
func Install(ctx context.Context, ac *Context, opts InstallOptions) (InstallResult, error) {
	return runInstallTxn(ctx, ac, opts, nil, nil)
}

// Add declares a dependency in memory and installs (manifest written at commit).
func Add(ctx context.Context, ac *Context, spec string, opts AddOptions) (InstallResult, error) {
	name, _ := parsePackageSpec(spec)
	if name == "" {
		return InstallResult{}, apperr.New(apperr.Usage, "app.add", spec, "invalid package name")
	}
	inst := opts.Install
	inst.WriteManifest = true
	inst.AddSpec = spec
	inst.AddDev = opts.Dev
	inst.AddSaveExact = opts.SaveExact
	return runInstallTxn(ctx, ac, inst, nil, prepareAddDependency)
}

// Remove deletes a dependency in memory and reinstalls (manifest written at commit).
func Remove(ctx context.Context, ac *Context, name string, opts InstallOptions) (InstallResult, error) {
	fields := []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"}
	edit := func(p *project.Project) error {
		var removed bool
		for _, field := range fields {
			if err := p.Doc.RemoveDependency(field, name); err != nil {
				if apperr.CodeOf(err) == apperr.NotFound {
					continue
				}
				return err
			}
			removed = true
			break
		}
		if !removed {
			return apperr.New(apperr.NotFound, "app.remove", name, "dependency not found")
		}
		return nil
	}
	opts.WriteManifest = true
	return runInstallTxn(ctx, ac, opts, edit, nil)
}

// FormatInstallSummary returns a human-readable install summary line.
func FormatInstallSummary(r InstallResult) string {
	line := fmt.Sprintf("added %d, removed %d, changed %d (%d packages)", r.Added, r.Removed, r.Changed, r.Packages)
	if r.CleanupIncomplete {
		line += "\nInstallation committed, but transaction cleanup is incomplete. Run m recover to clear stale transaction metadata."
		for _, w := range r.CleanupWarnings {
			line += "\n  " + w
		}
	}
	if r.RolledBack && r.RecoveryRequired {
		line += "\nRollback completed with cleanup warnings. Run m recover if stale transaction metadata remains."
	}
	return line
}

func populateCleanupResult(res *InstallResult, finish transaction.FinishResult) {
	if !finish.HasCriticalCleanupFailure() {
		return
	}
	res.Committed = finish.Committed
	res.CleanupIncomplete = true
	res.RecoveryRequired = true
	res.CleanupWarningCodes = append(res.CleanupWarningCodes, finish.CleanupWarningCodes...)
	for _, w := range finish.CleanupWarnings {
		if w != nil {
			res.CleanupWarnings = append(res.CleanupWarnings, w.Error())
		}
	}
}

func installCleanupIncompleteError(res InstallResult) error {
	msg := "Installation committed, but transaction cleanup is incomplete. Run m recover to clear stale transaction metadata."
	return apperr.New(apperr.Transaction, "app.install.cleanup", "", msg)
}

// ponytail: fetch/link helpers remain here; commit path is install_txn.go
func parsePackageSpec(spec string) (name, version string) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", ""
	}
	if !strings.HasPrefix(spec, "@") {
		if i := strings.LastIndex(spec, "@"); i > 0 {
			return spec[:i], spec[i+1:]
		}
		return spec, "latest"
	}
	slash := strings.Index(spec, "/")
	if slash < 0 {
		return spec, "latest"
	}
	rest := spec[slash+1:]
	if i := strings.Index(rest, "@"); i >= 0 {
		return spec[:slash+1+i], rest[i+1:]
	}
	return spec, "latest"
}
