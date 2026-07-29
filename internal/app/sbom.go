package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/sbom"
)

// SBOMFormat selects the SBOM serialization.
type SBOMFormat string

const (
	SBOMCycloneDX SBOMFormat = "cyclonedx"
	SBOMSPDX      SBOMFormat = "spdx"
)

// ExportSBOMOptions configures m sbom.
type ExportSBOMOptions struct {
	Format         SBOMFormat
	RedactInternal bool
	RedactPattern  string
}

// ExportSBOM reads the project lock graph and emits an SBOM document.
func ExportSBOM(ctx context.Context, ac *Context, opts ExportSBOMOptions) ([]byte, error) {
	if ac == nil {
		return nil, apperr.New(apperr.Internal, "app.sbom", "", "missing app context")
	}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return nil, err
	}
	g, err := LoadInstalledGraph(ctx, ac, proj)
	if err != nil {
		return nil, err
	}
	licenses := readInstalledLicenses(proj.Root, g)
	sbomOpts := sbom.SBOMOptions{
		RedactInternal: opts.RedactInternal,
		RedactPattern:  opts.RedactPattern,
		Licenses:       licenses,
		ProjectName:    projectDisplayName(proj, g),
	}
	switch opts.Format {
	case "", SBOMCycloneDX:
		return sbom.ExportCycloneDX(g, sbomOpts)
	case SBOMSPDX:
		return sbom.ExportSPDX(g, sbomOpts)
	default:
		return nil, apperr.New(apperr.Usage, "app.sbom", string(opts.Format), "format must be cyclonedx or spdx")
	}
}

func projectDisplayName(proj *project.Project, g *graph.Graph) string {
	if proj != nil && proj.Normalized != nil && proj.Normalized.Name != "" {
		return proj.Normalized.Name
	}
	for _, im := range g.Importers {
		if im.ID == graph.RootImporter && im.Name != "" {
			return im.Name
		}
	}
	return "project"
}

func readInstalledLicenses(projRoot string, g *graph.Graph) map[string]string {
	if g == nil || projRoot == "" {
		return nil
	}
	nmRoot := filepath.Join(projRoot, "node_modules")
	out := make(map[string]string, len(g.Packages))
	for _, pkg := range g.Packages {
		key := pkg.ID.Key()
		path := packageJSONPath(nmRoot, pkg.ID.Name)
		if lic := readPackageLicense(path); lic != "" {
			out[key] = lic
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func packageJSONPath(nmRoot, name string) string {
	parts := strings.Split(name, "/")
	all := append([]string{nmRoot}, parts...)
	all = append(all, "package.json")
	return filepath.Join(all...)
}

func readPackageLicense(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var doc struct {
		License json.RawMessage `json:"license"`
	}
	if err := json.Unmarshal(data, &doc); err != nil || len(doc.License) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(doc.License, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var obj struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(doc.License, &obj); err == nil {
		return strings.TrimSpace(obj.Type)
	}
	return ""
}
