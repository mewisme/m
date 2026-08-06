package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/linker/planner"
	"github.com/mewisme/mew/internal/node"
	runtimepkg "github.com/mewisme/mew/internal/runtime"
	"github.com/mewisme/mew/internal/transaction"
)

const DoctorReportSchemaVersion = 1

// DoctorCheckStatus is ok, warn, or fail.
type DoctorCheckStatus string

const (
	DoctorStatusOK   DoctorCheckStatus = "ok"
	DoctorStatusWarn DoctorCheckStatus = "warn"
	DoctorStatusFail DoctorCheckStatus = "fail"
)

// DoctorCheck is one health check result.
type DoctorCheck struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

// DoctorReport is the JSON schema v1 output for m doctor.
type DoctorReport struct {
	SchemaVersion int           `json:"schemaVersion"`
	CheckedAt     string        `json:"checkedAt"`
	Checks        []DoctorCheck `json:"checks"`
	OK            bool          `json:"ok"`
}

// DoctorOptions tunes m doctor.
type DoctorOptions struct {
	Strict bool
}

// FilesystemProbeReport is the result of m development doctor filesystem.
type FilesystemProbeReport struct {
	StoreRoot string
	DestRoot  string
	Caps      planner.Capabilities
}

// Doctor runs PM and project health checks for end users.
func Doctor(ctx context.Context, ac *Context, opts DoctorOptions) (DoctorReport, error) {
	var rep DoctorReport
	if err := ctx.Err(); err != nil {
		return rep, err
	}
	rep.SchemaVersion = DoctorReportSchemaVersion
	rep.CheckedAt = time.Now().UTC().Format(time.RFC3339)

	rep.Checks = append(rep.Checks, doctorCheckProject(ctx, ac))
	rep.Checks = append(rep.Checks, doctorCheckConfig(ac))
	rep.Checks = append(rep.Checks, doctorCheckCacheStore(ctx, ac)...)
	rep.Checks = append(rep.Checks, doctorCheckLock(ctx, ac))
	rep.Checks = append(rep.Checks, doctorCheckFilesystem(ctx, ac))
	rep.Checks = append(rep.Checks, doctorCheckTxn(ctx, ac))
	rep.Checks = append(rep.Checks, doctorCheckNode())

	rep.OK = !reportHasStatus(rep, DoctorStatusFail)
	if opts.Strict && reportHasStatus(rep, DoctorStatusWarn) {
		rep.OK = false
	}
	return rep, nil
}

func reportHasStatus(rep DoctorReport, want DoctorCheckStatus) bool {
	for _, c := range rep.Checks {
		if DoctorCheckStatus(c.Status) == want {
			return true
		}
	}
	return false
}

func doctorCheckProject(ctx context.Context, ac *Context) DoctorCheck {
	check := DoctorCheck{ID: "project"}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = err.Error()
		check.Remediation = "run from a directory with a readable package.json"
		return check
	}
	check.Status = string(DoctorStatusOK)
	check.Message = fmt.Sprintf("package.json readable at %s", proj.Root)
	return check
}

func doctorCheckConfig(ac *Context) DoctorCheck {
	check := DoctorCheck{ID: "config"}
	if ac == nil || ac.Config == nil {
		check.Status = string(DoctorStatusFail)
		check.Message = "configuration not loaded"
		check.Remediation = "fix m.jsonc / MEW_* settings and retry"
		return check
	}
	check.Status = string(DoctorStatusOK)
	check.Message = "configuration loaded"
	return check
}

func doctorCheckCacheStore(ctx context.Context, ac *Context) []DoctorCheck {
	if ac == nil || ac.Config == nil {
		return []DoctorCheck{
			{ID: "cache", Status: string(DoctorStatusFail), Message: "configuration not loaded"},
			{ID: "store", Status: string(DoctorStatusFail), Message: "configuration not loaded"},
		}
	}
	cacheRoot := config.CacheRoot(ac.Config)
	storeRoot, err := config.StoreRoot(ac.Config)
	if err != nil {
		return []DoctorCheck{{
			ID: "store", Status: string(DoctorStatusFail), Message: err.Error(),
			Remediation: "set store.dir or MEW_STORE_DIR to a writable path",
		}}
	}
	return []DoctorCheck{
		doctorWritableCheck(ctx, "cache", cacheRoot, "set cache.dir or MEW_CACHE_DIR to a writable path"),
		doctorWritableCheck(ctx, "store", storeRoot, "set store.dir or MEW_STORE_DIR to a writable path"),
	}
}

func doctorWritableCheck(ctx context.Context, id, dir, remediation string) DoctorCheck {
	check := DoctorCheck{ID: id}
	if err := ctx.Err(); err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = err.Error()
		return check
	}
	if err := probeWritable(dir); err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = fmt.Sprintf("%s not writable: %v", dir, err)
		check.Remediation = remediation
		return check
	}
	check.Status = string(DoctorStatusOK)
	check.Message = fmt.Sprintf("%s writable", dir)
	return check
}

func probeWritable(dir string) error {
	if dir == "" {
		return fmt.Errorf("empty path")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	probe := filepath.Join(dir, ".mew-doctor-probe")
	f, err := os.Create(probe)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Remove(probe)
}

func doctorCheckLock(ctx context.Context, ac *Context) DoctorCheck {
	check := DoctorCheck{ID: "lock"}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = err.Error()
		return check
	}
	path := LockPath(proj)
	if _, err := os.Stat(path); err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = fmt.Sprintf("lockfile missing: %s", path)
		check.Remediation = "run m install or m lock migrate to create a lockfile"
		return check
	}
	if _, err := ValidateIncumbentLock(ctx, ac, ValidateLockOptions{}); err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = err.Error()
		check.Remediation = "run m lock validate for details; regenerate with a supported package manager"
		return check
	}
	check.Status = string(DoctorStatusOK)
	check.Message = fmt.Sprintf("%s validated", filepath.Base(path))
	return check
}

func doctorCheckFilesystem(ctx context.Context, ac *Context) DoctorCheck {
	check := DoctorCheck{ID: "filesystem"}
	probe, err := DoctorFilesystem(ctx, ac)
	if err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = err.Error()
		check.Remediation = "ensure store and node_modules paths are accessible"
		return check
	}
	if !filesystemCapsUsable(probe.Caps) {
		check.Status = string(DoctorStatusWarn)
		check.Message = fmt.Sprintf("limited link support (hardlink=%v symlink=%v junction=%v); installs may copy files",
			probe.Caps.Hardlink, probe.Caps.Symlink, probe.Caps.Junction)
		check.Remediation = "see m development doctor filesystem for probe details"
		return check
	}
	check.Status = string(DoctorStatusOK)
	check.Message = fmt.Sprintf("link probe ok (store=%s)", probe.StoreRoot)
	return check
}

func filesystemCapsUsable(caps planner.Capabilities) bool {
	return caps.Hardlink || caps.Symlink || caps.Junction || caps.Reflink
}

func doctorCheckTxn(ctx context.Context, ac *Context) DoctorCheck {
	check := DoctorCheck{ID: "transaction"}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = err.Error()
		return check
	}
	if err := ctx.Err(); err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = err.Error()
		return check
	}
	txns, err := transaction.ScanIncompleteTxns(proj.Root)
	if err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = err.Error()
		return check
	}
	if len(txns) == 0 {
		check.Status = string(DoctorStatusOK)
		check.Message = "no incomplete transactions"
		return check
	}
	ids := make([]string, 0, len(txns))
	for _, t := range txns {
		ids = append(ids, fmt.Sprintf("%s(%s)", t.ID, t.State))
	}
	check.Status = string(DoctorStatusWarn)
	check.Message = fmt.Sprintf("incomplete transaction journals: %s", strings.Join(ids, ", "))
	check.Remediation = "run m recover to roll back or discard stale journals"
	return check
}

func doctorCheckNode() DoctorCheck {
	check := DoctorCheck{ID: "node"}
	path, err := exec.LookPath("node")
	if err != nil {
		check.Status = string(DoctorStatusWarn)
		check.Message = "node not found on PATH"
		check.Remediation = "install Node.js for script execution (runner support is MVP 0040)"
		return check
	}
	check.Status = string(DoctorStatusOK)
	check.Message = path
	return check
}

// DoctorExitError reports failed health checks to the CLI layer.
func DoctorExitError(rep DoctorReport) error {
	if rep.OK {
		return nil
	}
	return apperr.New(apperr.Integrity, "doctor", "", "health check failed")
}

// DoctorRuntime runs runtime-specific health checks: Node installation,
// capabilities, transform cache, and runtime asset integrity.
func DoctorRuntime(ctx context.Context, ac *Context, opts DoctorOptions) (DoctorReport, error) {
	var rep DoctorReport
	if err := ctx.Err(); err != nil {
		return rep, err
	}
	rep.SchemaVersion = DoctorReportSchemaVersion
	rep.CheckedAt = time.Now().UTC().Format(time.RFC3339)

	rep.Checks = append(rep.Checks, doctorCheckNodeCapabilities(ctx))
	if ac != nil && ac.Config != nil {
		rep.Checks = append(rep.Checks, doctorCheckRuntimeCache(ctx, ac))
	}

	rep.OK = !reportHasStatus(rep, DoctorStatusFail)
	if opts.Strict && reportHasStatus(rep, DoctorStatusWarn) {
		rep.OK = false
	}
	return rep, nil
}

func doctorCheckNodeCapabilities(ctx context.Context) DoctorCheck {
	check := DoctorCheck{ID: "node-capabilities"}
	inst, err := node.Discover(ctx, node.Request{})
	if err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = fmt.Sprintf("node discovery failed: %v", err)
		check.Remediation = "install Node.js 18+ (m requires module-register, import-preload, require-preload)"
		return check
	}
	if inst == nil {
		check.Status = string(DoctorStatusFail)
		check.Message = "node not found"
		check.Remediation = "install Node.js 18+"
		return check
	}
	capSet := make(map[string]bool, len(inst.Capabilities))
	for _, c := range inst.Capabilities {
		capSet[c] = true
	}
	required := []string{"require-preload", "import-preload", "module-register"}
	var missing []string
	for _, c := range required {
		if !capSet[c] {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		check.Status = string(DoctorStatusFail)
		check.Message = fmt.Sprintf("Node %s at %s missing capabilities: %s", inst.NormalizedVersion, inst.ExePath, strings.Join(missing, ", "))
		check.Remediation = "install Node.js 18+ or a newer LTS release"
		return check
	}
	check.Status = string(DoctorStatusOK)
	check.Message = fmt.Sprintf("Node %s at %s (capabilities: %s)", inst.NormalizedVersion, inst.ExePath, strings.Join(inst.Capabilities, ", "))
	return check
}

func doctorCheckRuntimeCache(ctx context.Context, ac *Context) DoctorCheck {
	check := DoctorCheck{ID: "runtime-cache"}
	cacheDir, err := runtimepkg.CacheDir(ac.Config)
	if err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = fmt.Sprintf("cache dir resolution failed: %v", err)
		return check
	}
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		check.Status = string(DoctorStatusWarn)
		check.Message = fmt.Sprintf("runtime cache not populated at %s", cacheDir)
		check.Remediation = "run any TypeScript file with m to populate the cache"
		return check
	}
	if err := runtimepkg.VerifyCache(ac.Config); err != nil {
		check.Status = string(DoctorStatusFail)
		check.Message = fmt.Sprintf("runtime cache verification failed: %v", err)
		check.Remediation = "run m cache explain for details; delete the cache directory to force re-extraction"
		return check
	}
	check.Status = string(DoctorStatusOK)
	check.Message = fmt.Sprintf("runtime cache valid at %s", cacheDir)
	return check
}

// FormatDoctorReport renders human-readable doctor output.
func FormatDoctorReport(rep DoctorReport) string {
	var b strings.Builder
	for _, c := range rep.Checks {
		fmt.Fprintf(&b, "check=%s status=%s message=%s", c.ID, c.Status, c.Message)
		if c.Remediation != "" {
			fmt.Fprintf(&b, " remediation=%s", c.Remediation)
		}
		b.WriteByte('\n')
	}
	if rep.OK {
		b.WriteString("doctor=ok\n")
	} else {
		b.WriteString("doctor=failed\n")
	}
	return b.String()
}

// DoctorFilesystem probes link capabilities between store and node_modules.
func DoctorFilesystem(ctx context.Context, ac *Context) (FilesystemProbeReport, error) {
	var rep FilesystemProbeReport
	if err := ctx.Err(); err != nil {
		return rep, err
	}
	if ac == nil || ac.Config == nil {
		return rep, apperr.New(apperr.Internal, "app.doctor.filesystem", "", "missing app context")
	}
	storeRoot, err := config.StoreRoot(ac.Config)
	if err != nil {
		return rep, err
	}
	rep.StoreRoot = storeRoot
	rep.DestRoot = "node_modules"
	if p, err := OpenProject(ctx, ac); err == nil {
		rep.DestRoot = filepath.Join(p.Root, "node_modules")
	}
	rep.Caps, err = planner.ProbeCached(config.CacheRoot(ac.Config), rep.StoreRoot, rep.DestRoot)
	return rep, err
}

// FormatFilesystemProbe returns human-readable probe output.
func FormatFilesystemProbe(rep FilesystemProbeReport) string {
	return fmt.Sprintf("src=%s\ndest=%s\nsameVolume=%v hardlink=%v reflink=%v symlink=%v junction=%v\n",
		rep.StoreRoot, rep.DestRoot,
		rep.Caps.SameVolume, rep.Caps.Hardlink, rep.Caps.Reflink, rep.Caps.Symlink, rep.Caps.Junction)
}
