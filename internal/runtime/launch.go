package runtime

import (
	"context"
	"errors"
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

	plan := &LaunchPlan{
		NodeExe:          nodeInst.ExePath,
		NodeVersion:      nodeInst.NormalizedVersion,
		NodeCapabilities: nodeInst.Capabilities,
		Entrypoint:       req.Entrypoint,
		AppArgs:          append([]string(nil), req.AppArgs...),
		ZeroAugmentation: req.AugmentationMode == AugmentNone,
	}

	// Apply launch contribution from app-level orchestrator.
	if req.Contribution != nil {
		plan.CleanupHook = req.Contribution.CleanupHook
		for _, pa := range req.Contribution.ExtraPreloads {
			plan.PreloadAssets = append(plan.PreloadAssets, pa)
		}
		plan.EnvChanges = append(plan.EnvChanges, req.Contribution.ExtraEnv...)
	}

	if req.AugmentationMode != AugmentNone {
		// Verify cached assets before use; corrupt entries are deleted.
		// After verification failure, EnsureAssets will re-extract missing files.
		_ = VerifyCache(eff)
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
			plan.PreloadAssets = append(plan.PreloadAssets, PreloadAsset{
				Path:       p,
				ModuleType: entry.ModuleType,
			})
		}
	}

	// Build argv
	plan.NodeArgv = BuildArgv(plan, req.NodeV8Args)
	return plan, nil
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
// Order: node exe -> user V8 flags -> preload/import flags -> entrypoint -> app args
func BuildArgv(plan *LaunchPlan, v8Args []string) []string {
	argv := make([]string, 0, 4+len(v8Args)+len(plan.PreloadAssets)*2+1+len(plan.AppArgs))
	argv = append(argv, plan.NodeExe)

	// user V8/Node flags
	argv = append(argv, v8Args...)

	if !plan.ZeroAugmentation {
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
	// Convert backslashes to forward slashes.
	p = strings.ReplaceAll(p, "\\", "/")
	// Remove leading slash for the drive letter format.
	p = strings.TrimPrefix(p, "/")
	// Encode spaces and other special chars in path segments (ponytail: minimal).
	p = strings.ReplaceAll(p, " ", "%20")
	return "file:///" + p
}

// Launch starts a Node process from a fully resolved plan.
func Launch(ctx context.Context, plan *LaunchPlan, req LaunchRequest) error {
	if plan == nil {
		return apperr.New(apperr.RuntimeInvocation, "runtime.launch", "", "nil plan")
	}

	supervisor := process.NewExecSupervisor()
	spec := process.Spec{
		Path:   plan.NodeArgv[0],
		Args:   plan.NodeArgv[1:],
		Dir:    req.WorkingDir,
		Env:    buildEnv(req.EnvOverlay, plan.EnvChanges),
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
