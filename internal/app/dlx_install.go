package app

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/lifecycle"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/runner/dlx"
)

func buildDLXEnvironment(ctx context.Context, ac *Context, opts DLXOptions, resolved *dlxResolveResult, envDir string) error {
	if resolved == nil || resolved.Resolution == nil {
		return apperr.New(apperr.Internal, "app.dlx.install", "", "missing resolution")
	}
	mxRoot := config.MXCacheDir(ac.Config)
	staging := dlx.StagingDir(mxRoot, resolved.Identity.Digest(), resolved.TxnID)
	if err := os.RemoveAll(staging); err != nil {
		return apperr.Wrap(apperr.IO, "app.dlx.install", staging, err)
	}
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "app.dlx.install", staging, err)
	}
	proj, err := project.Open(ctx, resolved.EphemeralDir)
	if err != nil {
		return err
	}
	extractDir := filepath.Join(staging, ".extract")
	fetchOut, err := fetchPackages(ctx, ac, proj, resolved.Resolution.Graph, resolved.Resolution.Extensions, extractDir, false, nil)
	if err != nil {
		return err
	}
	linkerMode := config.String(ac.Config, "install.linker", "auto")
	stageNM := filepath.Join(staging, "node_modules")
	if err := os.MkdirAll(stageNM, 0o755); err != nil {
		return err
	}
	lnk := newLinker(linkerMode, linkerOpts{NodeModules: stageNM, ExtractDirs: fetchOut.Extracts})
	linkPlan, err := lnk.Plan(ctx, resolved.Resolution.Graph)
	if err != nil {
		return err
	}
	if err := lnk.Apply(ctx, linkPlan); err != nil {
		return err
	}
	genBind, err := publishLinkBinMetadata(staging, linkPlan, linkerMode, resolved.Identity.GraphDigest)
	if err != nil {
		return err
	}
	if err := WriteGenerationBinding(staging, genBind); err != nil {
		return err
	}
	if err := writeResolvedLock(filepath.Join(staging, "m.lock"), ac, resolved.Resolution); err != nil {
		return err
	}
	if lifecycle.Enabled(ac.Config) {
		_ = runLifecyclePhase(ctx, ac, proj, InstallOptions{IgnoreScripts: true}, stageNM, resolved.Resolution.Graph, linkPlan)
	}
	if err := validateDLXUsability(stageNM, opts, resolved); err != nil {
		return err
	}
	ready := dlx.ReadyMarker{
		EnvironmentDigest:          resolved.Identity.Digest(),
		GraphDigest:                resolved.Identity.GraphDigest,
		TargetPlatformFingerprint:  resolved.Identity.TargetPlatformFingerprint,
		NodeFingerprint:            resolved.Identity.NodeFingerprint,
		LifecyclePolicyFingerprint: resolved.Identity.LifecyclePolicyFingerprint,
	}
	return dlx.PublishEnvironment(staging, envDir, ready)
}

func validateDLXUsability(stageNM string, opts DLXOptions, resolved *dlxResolveResult) error {
	command, _, err := selectDLXCommand(opts, resolved)
	if err != nil {
		return err
	}
	shim := filepath.Join(stageNM, ".bin", command)
	if runtime.GOOS == "windows" {
		shim += ".cmd"
	}
	if _, err := os.Stat(shim); err != nil {
		return apperr.New(apperr.NotFound, "app.dlx.install", command, "binary not materialized")
	}
	return nil
}
