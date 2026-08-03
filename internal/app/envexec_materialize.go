package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lifecycle"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/runner/envexec"
)

type appFrozenMaterializer struct {
	ac *Context
}

func (m appFrozenMaterializer) Materialize(ctx context.Context, spec envexec.FrozenEnvironmentSpec, finalDir string) error {
	g, ok := spec.Graph.(*graph.Graph)
	if !ok || g == nil {
		return apperr.New(apperr.Internal, "app.materialize", "", "missing graph")
	}
	if err := os.RemoveAll(finalDir); err != nil {
		return apperr.Wrap(apperr.IO, "app.materialize", finalDir, err)
	}
	if err := os.MkdirAll(finalDir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "app.materialize", finalDir, err)
	}
	if err := os.WriteFile(filepath.Join(finalDir, "package.json"), spec.Manifest, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "app.materialize", finalDir, err)
	}
	if err := os.WriteFile(filepath.Join(finalDir, "m.lock"), spec.LockSnapshot, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "app.materialize", finalDir, err)
	}
	proj, err := project.Open(ctx, finalDir)
	if err != nil {
		return err
	}
	extractDir := filepath.Join(finalDir, ".extract")
	preExtracts := spec.PrelinkedExtracts
	if preExtracts == nil {
		preExtracts = map[string]string{}
	}
	fetchOut, err := fetchPackages(ctx, m.ac, proj, g, nil, extractDir, false, preExtracts, nil)
	if err != nil {
		return err
	}
	linkerMode := spec.LinkerMode
	if linkerMode == "" {
		linkerMode = "auto"
	}
	stageNM := filepath.Join(finalDir, "node_modules")
	if err := os.MkdirAll(stageNM, 0o755); err != nil {
		return err
	}
	lnk := newLinker(linkerMode, linkerOpts{NodeModules: stageNM, ExtractDirs: fetchOut.Extracts})
	linkPlan, err := lnk.Plan(ctx, g)
	if err != nil {
		return err
	}
	if err := lnk.Apply(ctx, linkPlan); err != nil {
		return err
	}
	genID := spec.Identity.GraphDigest
	if genID == "" {
		genID = spec.Identity.MaterialDigest
	}
	genBind, err := publishLinkBinMetadata(finalDir, linkPlan, linkerMode, genID)
	if err != nil {
		return err
	}
	if err := WriteGenerationBinding(finalDir, genBind); err != nil {
		return err
	}
	if spec.LifecyclePolicy != envexec.LifecycleForbidden && lifecycle.Enabled(m.ac.Config) {
		// intentional: ephemeral env; a failed install script must not abort materialization
		_ = runLifecyclePhase(ctx, m.ac, proj, InstallOptions{IgnoreScripts: true}, stageNM, g, linkPlan, "", nil)
	}
	return nil
}
