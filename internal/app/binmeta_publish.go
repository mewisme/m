package app

import (
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/binmeta"
	"github.com/mewisme/mew/internal/linker"
)

// publishLinkBinMetadata writes generation-bound bins metadata for each node_modules tree in plan.
func publishLinkBinMetadata(stageRoot string, plan *linker.Plan, linkerMode, generationID string) (GenerationBinding, error) {
	var empty GenerationBinding
	if plan == nil || generationID == "" {
		return empty, nil
	}
	byNM := map[string][]linker.BinSource{}
	for _, src := range plan.Bins {
		nm := src.NodeModules
		if nm == "" {
			nm = plan.NodeModules
		}
		if nm == "" {
			continue
		}
		byNM[nm] = append(byNM[nm], src)
	}
	if len(byNM) == 0 && plan.NodeModules != "" {
		byNM[plan.NodeModules] = nil
	}
	mode := binmetaLayoutMode(linkerMode, plan.LayoutMode)
	var rootBind GenerationBinding
	for nm, sources := range byNM {
		importer := importerIdentityFromNodeModules(stageRoot, nm)
		if err := binmeta.Publish(binmeta.PublishInput{
			NodeModules:      nm,
			ImporterIdentity: importer,
			GenerationID:     generationID,
			LayoutMode:       mode,
			Sources:          sources,
		}); err != nil {
			return empty, apperr.Wrap(apperr.Install, "app.binmeta", nm, err)
		}
		if importer == "." {
			doc, readErr := binmeta.Read(nm)
			if readErr != nil {
				return empty, apperr.Wrap(apperr.Install, "app.binmeta", nm, readErr)
			}
			rootBind = GenerationBinding{GenerationID: generationID, Fingerprint: doc.Fingerprint}
		}
	}
	if rootBind.GenerationID == "" {
		rootBind.GenerationID = generationID
	}
	return rootBind, nil
}

func binmetaLayoutMode(linkerMode, planMode string) binmeta.LayoutMode {
	switch strings.TrimSpace(planMode) {
	case "isolated":
		return binmeta.LayoutIsolated
	case "pnp":
		return binmeta.LayoutPnP
	}
	switch linkerMode {
	case "isolated":
		return binmeta.LayoutIsolated
	default:
		return binmeta.LayoutHoisted
	}
}

func importerIdentityFromNodeModules(stageRoot, nodeModules string) string {
	rel, err := filepath.Rel(stageRoot, filepath.Clean(nodeModules))
	if err != nil {
		return "."
	}
	rel = filepath.ToSlash(rel)
	switch {
	case rel == "node_modules":
		return "."
	case strings.HasSuffix(rel, "/node_modules"):
		return strings.TrimSuffix(rel, "/node_modules")
	default:
		return "."
	}
}
