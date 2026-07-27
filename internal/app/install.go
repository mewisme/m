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
	Added                        int        `json:"added"`
	Removed                      int        `json:"removed"`
	Changed                      int        `json:"changed"`
	Packages                     int        `json:"packages"`
	Plan                         *plan.Plan `json:"plan,omitempty"`
	Committed                    bool       `json:"committed,omitempty"`
	RolledBack                   bool       `json:"rolledBack,omitempty"`
	RecoveryRequired             bool       `json:"recoveryRequired,omitempty"`
	CleanupIncomplete            bool       `json:"cleanupIncomplete,omitempty"`
	TransactionCleanupIncomplete bool       `json:"transactionCleanupIncomplete,omitempty"`
	StoreCleanupIncomplete       bool       `json:"storeCleanupIncomplete,omitempty"`
	StoreMaintenanceRequired     bool       `json:"storeMaintenanceRequired,omitempty"`
	CleanupWarningCodes          []string   `json:"cleanupWarningCodes,omitempty"`
	CleanupWarnings              []string   `json:"cleanupWarnings,omitempty"`
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

// AddInSession declares a dependency using an existing mutation session (caller holds the lock).
func AddInSession(ctx context.Context, sess *MutationSession, spec string, opts AddOptions) (InstallResult, error) {
	name, _ := parsePackageSpec(spec)
	if name == "" {
		return InstallResult{}, apperr.New(apperr.Usage, "app.add", spec, "invalid package name")
	}
	inst := opts.Install
	inst.WriteManifest = true
	inst.AddSpec = spec
	inst.AddDev = opts.Dev
	inst.AddSaveExact = opts.SaveExact
	return runInstallInSession(ctx, sess, inst, nil, prepareAddDependency)
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
	seen := map[string]bool{}
	line := fmt.Sprintf("added %d, removed %d, changed %d (%d packages)", r.Added, r.Removed, r.Changed, r.Packages)
	if r.Committed && (r.TransactionCleanupIncomplete || r.RecoveryRequired) {
		line += "\nInstallation committed, but transaction cleanup is incomplete. Run m recover to clear stale transaction metadata."
		for _, w := range formatCleanupSection(r, criticalTxnCleanupCodes, seen) {
			line += "\n  " + w
		}
	} else if r.RolledBack && (r.TransactionCleanupIncomplete || r.RecoveryRequired) {
		line += "\nRollback completed with cleanup warnings. Run m recover if stale transaction metadata remains."
		for _, w := range formatCleanupSection(r, criticalTxnCleanupCodes, seen) {
			line += "\n  " + w
		}
	}
	if r.Committed && r.CleanupIncomplete {
		line += "\nInstallation committed with a non-critical cleanup warning."
		for _, w := range formatCleanupSection(r, nonCriticalTxnCleanupCodes, seen) {
			line += "\n  " + w
		}
	}
	if r.StoreCleanupIncomplete || r.StoreMaintenanceRequired {
		line += "\nStore cleanup is incomplete. Run m store status for details."
		for _, w := range formatCleanupSection(r, storeCleanupCodes, seen) {
			line += "\n  " + w
		}
	}
	for _, w := range formatUnknownCleanupSection(r, seen) {
		line += "\n  " + w
	}
	return line
}

var (
	criticalTxnCleanupCodes = map[string]bool{
		cleanupCodeTxnLockRelease:    true,
		cleanupCodeTxnCurrentCleanup: true,
	}
	nonCriticalTxnCleanupCodes = map[string]bool{
		"finish_hook":    true,
		"txn_dir_remove": true,
	}
	storeCleanupCodes = map[string]bool{
		cleanupCodeStoreImportLockRelease: true,
		cleanupCodeStoreIndexLockRelease:  true,
	}
)

func formatCleanupSection(r InstallResult, allowed map[string]bool, seen map[string]bool) []string {
	var out []string
	for i, code := range r.CleanupWarningCodes {
		if !allowed[code] {
			continue
		}
		msg := ""
		if i < len(r.CleanupWarnings) {
			msg = r.CleanupWarnings[i]
		}
		key := code + "\x00" + msg
		if seen[key] || msg == "" {
			continue
		}
		seen[key] = true
		out = append(out, msg)
	}
	return out
}

func formatUnknownCleanupSection(r InstallResult, seen map[string]bool) []string {
	var out []string
	for i, code := range r.CleanupWarningCodes {
		if criticalTxnCleanupCodes[code] || nonCriticalTxnCleanupCodes[code] || storeCleanupCodes[code] {
			continue
		}
		msg := ""
		if i < len(r.CleanupWarnings) {
			msg = r.CleanupWarnings[i]
		}
		key := code + "\x00" + msg
		if seen[key] || msg == "" {
			continue
		}
		seen[key] = true
		out = append(out, msg)
	}
	return out
}

func applyFetchOutcome(res *InstallResult, out FetchOutcome) {
	if res == nil {
		return
	}
	mergeStoreCleanupIntoResult(res, out.CleanupWarningCodes, out.CleanupWarnings)
	if out.StoreCleanupIncomplete {
		res.StoreCleanupIncomplete = true
	}
	if out.StoreMaintenanceRequired {
		res.StoreMaintenanceRequired = true
	}
}

func mergeStoreCleanupIntoResult(res *InstallResult, codes, warnings []string) {
	for i, code := range codes {
		msg := ""
		if i < len(warnings) {
			msg = warnings[i]
		}
		if cleanupPairContains(res.CleanupWarningCodes, res.CleanupWarnings, code, msg) {
			continue
		}
		res.CleanupWarningCodes = append(res.CleanupWarningCodes, code)
		res.CleanupWarnings = append(res.CleanupWarnings, msg)
	}
	for i := len(codes); i < len(warnings); i++ {
		msg := warnings[i]
		if cleanupPairContains(res.CleanupWarningCodes, res.CleanupWarnings, "", msg) {
			continue
		}
		res.CleanupWarnings = append(res.CleanupWarnings, msg)
	}
	if len(codes) > 0 || len(warnings) > 0 {
		res.StoreCleanupIncomplete = true
		res.StoreMaintenanceRequired = true
	}
}

func cleanupPairContains(codes, warnings []string, code, msg string) bool {
	for i, c := range codes {
		w := ""
		if i < len(warnings) {
			w = warnings[i]
		}
		if c == code && w == msg {
			return true
		}
	}
	if code == "" {
		for _, w := range warnings {
			if w == msg {
				return true
			}
		}
	}
	return false
}

func storeMaintenanceIncompleteError(res InstallResult) error {
	_ = res
	return apperr.New(apperr.Store, "app.install.store_cleanup", "",
		"Installation committed, but store cleanup is incomplete. Run m store status for details.")
}

func populateCleanupResult(res *InstallResult, finish transaction.FinishResult) {
	if !finish.HasCriticalCleanupFailure() {
		return
	}
	res.Committed = finish.Committed
	res.CleanupIncomplete = true
	res.TransactionCleanupIncomplete = true
	res.RecoveryRequired = true
	for i, w := range finish.CleanupWarnings {
		if w == nil {
			continue
		}
		code := ""
		if i < len(finish.CleanupWarningCodes) {
			code = finish.CleanupWarningCodes[i]
		}
		if transaction.CleanupCodeSeverity(code) != transaction.CleanupCritical {
			continue
		}
		res.CleanupWarningCodes = append(res.CleanupWarningCodes, code)
		res.CleanupWarnings = append(res.CleanupWarnings, w.Error())
	}
}

func populateWarningCleanup(res *InstallResult, finish transaction.FinishResult) {
	warnings := finish.WarningErrors()
	if len(warnings) == 0 {
		return
	}
	res.CleanupIncomplete = true
	for i, w := range finish.CleanupWarnings {
		if w == nil {
			continue
		}
		code := ""
		if i < len(finish.CleanupWarningCodes) {
			code = finish.CleanupWarningCodes[i]
		}
		if transaction.CleanupCodeSeverity(code) != transaction.CleanupWarning {
			continue
		}
		res.CleanupWarningCodes = append(res.CleanupWarningCodes, code)
		res.CleanupWarnings = append(res.CleanupWarnings, w.Error())
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
