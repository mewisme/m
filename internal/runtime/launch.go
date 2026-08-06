package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/node"
	"github.com/mewisme/mew/internal/process"
	"github.com/mewisme/mew/internal/runtime/assets"
)

// Plan resolves a LaunchRequest into a concrete LaunchPlan.
func Plan(ctx context.Context, req LaunchRequest, eff *config.Effective) (*LaunchPlan, error) {
	if req.Entrypoint == "" {
		return nil, apperr.New(apperr.RuntimeEntrypoint, "runtime.plan", "", "empty entrypoint")
	}

	nodeInst, err := node.Discover(ctx, node.Request{
		WorkingDir:        req.WorkingDir,
		ExplicitCandidate: "",
	})
	if err != nil {
		return nil, err
	}

	// Enforce required Node capabilities before building argv.
	if req.AugmentationMode != AugmentNone {
		if err := enforceCapabilities(nodeInst, req.Entrypoint); err != nil {
			return nil, err
		}
	}

	plan := &LaunchPlan{
		NodeExe:          nodeInst.ExePath,
		NodeVersion:      nodeInst.NormalizedVersion,
		NodeCapabilities: nodeInst.Capabilities,
		Entrypoint:       req.Entrypoint,
		AppArgs:          append([]string(nil), req.AppArgs...),
		ZeroAugmentation: req.AugmentationMode == AugmentNone,
	}

	// Resolve user-specified custom ESM loaders.
	if len(req.Loaders) > 0 && req.AugmentationMode != AugmentNone {
		for _, p := range req.Loaders {
			abs := p
			if !filepath.IsAbs(p) {
				abs = filepath.Join(req.WorkingDir, p)
			}
			plan.CustomLoaders = append(plan.CustomLoaders, PreloadAsset{
				Path:       abs,
				ModuleType: "esm",
			})
		}
	}

	// Apply launch contribution from app-level orchestrator.
	if req.Contribution != nil {
		plan.CleanupHook = req.Contribution.CleanupHook
		plan.PreloadAssets = append(plan.PreloadAssets, req.Contribution.ExtraPreloads...)
		plan.EnvChanges = append(plan.EnvChanges, req.Contribution.ExtraEnv...)
	}

	if req.AugmentationMode != AugmentNone {
		// Verify cached assets before use; corrupt entries are deleted.
		// VerifyCache only returns fatal errors (permission, I/O, manifest).
		// Missing or corrupt files are deleted so EnsureAssets re-extracts them.
		if err := VerifyCache(eff); err != nil {
			return nil, err
		}
		assetPaths, err := EnsureAssets(eff)
		if err != nil {
			return nil, err
		}
		m, err := assets.LoadManifest()
		if err != nil {
			return nil, err
		}
		needsTransform := isTypeScriptEntrypoint(req.Entrypoint)
		for _, entry := range m.AssetsSorted() {
			if !entry.Role.Injected() {
				continue
			}
			// Only inject transform-related assets for TypeScript entrypoints.
			if entry.Role == assets.RoleLoaderRegistration && !needsTransform {
				continue
			}
			p, ok := assetPaths[entry.Name]
			if !ok {
				continue
			}
			pa := PreloadAsset{
				Path:       p,
				ModuleType: entry.ModuleType,
			}
			if entry.Role == assets.RoleCredentialGrabber {
				plan.CredentialPreload = &pa
			} else {
				plan.PreloadAssets = append(plan.PreloadAssets, pa)
			}
		}
	}

	// Build argv
	plan.NodeArgv = BuildArgv(plan, req.NodeV8Args)
	return plan, nil
}

// enforceCapabilities verifies the Node installation supports required features.
func enforceCapabilities(inst *node.Installation, entrypoint string) error {
	capSet := make(map[string]bool, len(inst.Capabilities))
	for _, c := range inst.Capabilities {
		capSet[c] = true
	}
	required := []string{"require-preload", "import-preload"}
	if isTypeScriptEntrypoint(entrypoint) {
		required = append(required, "module-register")
	}
	for _, c := range required {
		if !capSet[c] {
			return apperr.New(apperr.RuntimeNodeUnsupported, "runtime.plan", inst.NormalizedVersion,
				fmt.Sprintf("Node %s lacks required capability %q", inst.NormalizedVersion, c))
		}
	}
	return nil
}

// isTypeScriptEntrypoint reports whether the entrypoint needs transform support.
func isTypeScriptEntrypoint(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".ts", ".mts", ".cts", ".tsx":
		return true
	}
	return false
}

// BuildArgv constructs the full Node argument vector.
// Order: node exe -> credential grabber --require -> user V8 flags
//
//	-> custom loader --import flags -> Mew preload flags -> entrypoint -> app args
//
// The credential grabber is placed before user V8 flags so it captures and
// strips transform credentials from process.env before any user --require,
// user --import, or NODE_OPTIONS preload runs.
func BuildArgv(plan *LaunchPlan, v8Args []string) []string {
	customSlots := len(plan.CustomLoaders) * 2
	credSlots := 0
	if !plan.ZeroAugmentation && plan.CredentialPreload != nil {
		credSlots = 2 // --require <path>
	}
	argv := make([]string, 0, 4+credSlots+len(v8Args)+customSlots+len(plan.PreloadAssets)*2+1+len(plan.AppArgs))
	argv = append(argv, plan.NodeExe)

	// Credential grabber runs FIRST — before any user preload.
	// Node processes --require from left to right; this must be the
	// first preload so credentials are stripped before user code.
	if !plan.ZeroAugmentation && plan.CredentialPreload != nil {
		argv = append(argv, "--require", plan.CredentialPreload.Path)
	}

	// user V8/Node flags
	argv = append(argv, v8Args...)

	if !plan.ZeroAugmentation {
		// Custom ESM loaders (--loader flag) injected BEFORE Mew preloads.
		// Custom loaders that register hooks via module.register() execute
		// first → their hooks run first in the chain. ts-loader hooks,
		// registered last, fill gaps.
		for _, pa := range plan.CustomLoaders {
			assetPath := pa.Path
			if runtime.GOOS == "windows" && filepath.IsAbs(assetPath) {
				assetPath = fileURL(assetPath)
			}
			argv = append(argv, "--import", assetPath)
		}

		for _, pa := range plan.PreloadAssets {
			assetPath := pa.Path
			switch pa.ModuleType {
			case "cjs":
				// --require works with native Windows paths; no URL conversion.
				argv = append(argv, "--require", assetPath)
			case "esm":
				// --import needs file:// URLs on Windows for ESM loader.
				if runtime.GOOS == "windows" && filepath.IsAbs(assetPath) {
					assetPath = fileURL(assetPath)
				}
				argv = append(argv, "--import", assetPath)
			}
		}
	}

	argv = append(argv, plan.Entrypoint)
	argv = append(argv, plan.AppArgs...)
	return argv
}

// fileURL converts an absolute Windows path to a file:// URL.
func fileURL(p string) string {
	// Use url.URL for standards-compliant file:// URL construction.
	// Handles drive letters, spaces, Unicode, and special chars correctly.
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	u := &url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	return u.String()
}

// MergeCleanupError merges launch and cleanup errors preserving primary type.
// When launch succeeds and cleanup fails: returns cleanup error.
// When both fail: preserves launch as primary, attaches cleanup.
// Child exit codes, cancellation, and timeout classification are preserved.
func MergeCleanupError(launchErr, cleanupErr error) error {
	return apperr.JoinCleanup(launchErr, cleanupErr)
}
func Launch(ctx context.Context, plan *LaunchPlan, req LaunchRequest) error {
	if plan == nil {
		return apperr.New(apperr.RuntimeInvocation, "runtime.launch", "", "nil plan")
	}

	childEnv := buildEnv(req.EnvOverlay, plan.EnvChanges)

	// Reject unsafe NODE_OPTIONS that would execute before credential isolation.
	if !plan.ZeroAugmentation {
		if err := ValidateNodeEnv(childEnv); err != nil {
			return err
		}
	}

	supervisor := process.NewExecSupervisor()
	spec := process.Spec{
		Path:   plan.NodeArgv[0],
		Args:   plan.NodeArgv[1:],
		Dir:    req.WorkingDir,
		Env:    childEnv,
		Stdin:  req.Stdio.Stdin,
		Stdout: req.Stdio.Stdout,
		Stderr: req.Stdio.Stderr,
	}

	h, err := supervisor.Start(ctx, spec)
	if err != nil {
		return apperr.Wrap(apperr.RuntimeNodeStart, "runtime.launch", plan.Entrypoint, err)
	}

	if err := supervisor.Wait(ctx, h); err != nil {
		var exitErr *process.ExitError
		if errors.As(err, &exitErr) && exitErr != nil {
			return &apperr.ExitStatus{Code: exitErr.Code, Err: err}
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return apperr.Wrap(apperr.Cancelled, "runtime.launch", plan.Entrypoint, context.Canceled)
		}
		return apperr.Wrap(apperr.RuntimeInvocation, "runtime.launch", plan.Entrypoint, err)
	}
	return nil
}

// ValidateNodeEnv rejects NODE_OPTIONS containing --require or --import.
// Node processes NODE_OPTIONS before CLI args, so user --require in
// NODE_OPTIONS would run before the credential grabber and could read
// transform credentials from process.env.
func ValidateNodeEnv(env []string) error {
	for _, kv := range env {
		const prefix = "NODE_OPTIONS="
		if !strings.HasPrefix(kv, prefix) {
			continue
		}
		val := kv[len(prefix):]
		if strings.Contains(val, "--require") || strings.Contains(val, "--import") {
			return apperr.New(apperr.Usage, "runtime.launch", "",
				"NODE_OPTIONS contains --require or --import, which would execute before credential isolation; pass these flags as CLI arguments instead")
		}
	}
	return nil
}

func buildEnv(envOverlay []string, planEnvChanges []string) []string {
	base := os.Environ()
	allOverlay := append([]string(nil), envOverlay...)
	allOverlay = append(allOverlay, planEnvChanges...)
	if len(allOverlay) == 0 {
		return base
	}
	overlay := make(map[string]string, len(allOverlay))
	for _, kv := range allOverlay {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				overlay[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	out := make([]string, 0, len(base)+len(overlay))
	for _, kv := range base {
		key := kv
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				key = kv[:i]
				break
			}
		}
		if _, replaced := overlay[key]; replaced {
			continue
		}
		out = append(out, kv)
	}
	for k, v := range overlay {
		out = append(out, k+"="+v)
	}
	return out
}
