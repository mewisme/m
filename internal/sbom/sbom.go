package sbom

import (
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/contentid"
	"github.com/mewisme/mew/internal/graph"
)

// SBOMOptions controls SBOM export behavior.
type SBOMOptions struct {
	RedactInternal bool
	RedactPattern  string
	Licenses       map[string]string // package key -> license expression
	ProjectName    string
	GeneratedAt    time.Time // zero => time.Now().UTC()
}

// Component is one package entry in an SBOM export.
type Component struct {
	Key       string
	Name      string
	Version   string
	PURL      string
	Integrity string
	License   string
	Redacted  bool
}

// ExportCycloneDX emits a minimal CycloneDX 1.5 JSON BOM from a lock graph.
func ExportCycloneDX(g *graph.Graph, opts SBOMOptions) ([]byte, error) {
	comps, err := componentsFromGraph(g, opts)
	if err != nil {
		return nil, err
	}
	ts := opts.GeneratedAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	project := opts.ProjectName
	if project == "" {
		project = projectNameFromGraph(g)
	}

	type hashEntry struct {
		Alg     string `json:"alg"`
		Content string `json:"content"`
	}
	type licenseEntry struct {
		License struct {
			ID string `json:"id"`
		} `json:"license"`
	}
	type componentEntry struct {
		Type     string         `json:"type"`
		Name     string         `json:"name"`
		Version  string         `json:"version,omitempty"`
		PURL     string         `json:"purl,omitempty"`
		Hashes   []hashEntry    `json:"hashes,omitempty"`
		Licenses []licenseEntry `json:"licenses,omitempty"`
	}
	type metadata struct {
		Timestamp string `json:"timestamp"`
		Component struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			Version string `json:"version,omitempty"`
		} `json:"component"`
	}
	type bom struct {
		BOMFormat   string           `json:"bomFormat"`
		SpecVersion string           `json:"specVersion"`
		Version     int              `json:"version"`
		Metadata    metadata         `json:"metadata"`
		Components  []componentEntry `json:"components"`
	}

	out := bom{
		BOMFormat:   "CycloneDX",
		SpecVersion: "1.5",
		Version:     1,
	}
	out.Metadata.Timestamp = ts.Format(time.RFC3339)
	out.Metadata.Component.Type = "application"
	out.Metadata.Component.Name = project

	for _, c := range comps {
		entry := componentEntry{
			Type:    "library",
			Name:    c.Name,
			Version: c.Version,
			PURL:    c.PURL,
		}
		if alg, hex, ok := integrityHash(c.Integrity); ok {
			entry.Hashes = []hashEntry{{Alg: alg, Content: hex}}
		}
		if lic := strings.TrimSpace(c.License); lic != "" {
			entry.Licenses = []licenseEntry{{License: struct {
				ID string `json:"id"`
			}{ID: lic}}}
		}
		out.Components = append(out.Components, entry)
	}

	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return nil, apperr.Wrap(apperr.Internal, "sbom.cyclonedx", "encode", err)
	}
	return []byte(buf.String()), nil
}

// ExportSPDX emits an SPDX 2.3 tag-value document from a lock graph.
func ExportSPDX(g *graph.Graph, opts SBOMOptions) ([]byte, error) {
	comps, err := componentsFromGraph(g, opts)
	if err != nil {
		return nil, err
	}
	ts := opts.GeneratedAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	project := opts.ProjectName
	if project == "" {
		project = projectNameFromGraph(g)
	}

	var b strings.Builder
	writeTag := func(key, value string) {
		if value == "" {
			return
		}
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(value)
		b.WriteByte('\n')
	}

	writeTag("SPDXVersion", "SPDX-2.3")
	writeTag("DataLicense", "CC0-1.0")
	writeTag("SPDXID", "SPDXRef-DOCUMENT")
	writeTag("DocumentName", project+"-sbom")
	writeTag("DocumentNamespace", "https://mew.is/sbom/"+url.PathEscape(project))
	writeTag("Creator", "Tool: Mew-m")
	writeTag("Created", ts.Format(time.RFC3339))
	b.WriteByte('\n')

	for i, c := range comps {
		spdxID := "SPDXRef-Package-" + itoa(i+1)
		writeTag("PackageName", c.Name)
		writeTag("SPDXID", spdxID)
		writeTag("VersionInfo", c.Version)
		writeTag("PackageDownloadLocation", "NOASSERTION")
		if c.PURL != "" {
			writeTag("ExternalRef", "PACKAGE-MANAGER purl "+c.PURL)
		}
		if lic := strings.TrimSpace(c.License); lic != "" {
			writeTag("PackageLicenseConcluded", lic)
			writeTag("PackageLicenseDeclared", lic)
		} else {
			writeTag("PackageLicenseConcluded", "NOASSERTION")
			writeTag("PackageLicenseDeclared", "NOASSERTION")
		}
		if sum := spdxChecksum(c.Integrity); sum != "" {
			writeTag("Checksum", sum)
		}
		b.WriteByte('\n')
	}

	return []byte(b.String()), nil
}

func componentsFromGraph(g *graph.Graph, opts SBOMOptions) ([]Component, error) {
	if g == nil {
		return nil, apperr.New(apperr.Lockfile, "sbom.components", "graph", "nil graph")
	}
	pattern, err := compileRedactPattern(opts.RedactPattern)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(g.Packages))
	byKey := make(map[string]graph.Package, len(g.Packages))
	for _, pkg := range g.Packages {
		key := pkg.ID.Key()
		keys = append(keys, key)
		byKey[key] = pkg
	}
	sort.Strings(keys)

	out := make([]Component, 0, len(keys))
	for _, key := range keys {
		pkg := byKey[key]
		name := pkg.ID.Name
		version := pkg.ID.Version
		lic := ""
		if opts.Licenses != nil {
			lic = opts.Licenses[key]
		}
		if shouldRedact(name, opts.RedactInternal, pattern) {
			out = append(out, Component{
				Key:      key,
				Name:     "[redacted]",
				Version:  "[redacted]",
				Redacted: true,
			})
			continue
		}
		out = append(out, Component{
			Key:       key,
			Name:      name,
			Version:   version,
			PURL:      npmPURL(name, version),
			Integrity: pkg.Integrity,
			License:   lic,
		})
	}
	return out, nil
}

func projectNameFromGraph(g *graph.Graph) string {
	for _, im := range g.Importers {
		if im.ID == graph.RootImporter && im.Name != "" {
			return im.Name
		}
	}
	if len(g.Importers) > 0 && g.Importers[0].Name != "" {
		return g.Importers[0].Name
	}
	return "project"
}

func npmPURL(name, version string) string {
	if name == "" || version == "" {
		return ""
	}
	return "pkg:npm/" + url.PathEscape(name) + "@" + version
}

func integrityHash(integrity string) (alg, hex string, ok bool) {
	id, err := contentid.ParseSRI(integrity)
	if err != nil {
		return "", "", false
	}
	alg = strings.ToUpper(id.Algo)
	switch alg {
	case "SHA256":
		alg = "SHA-256"
	case "SHA512":
		alg = "SHA-512"
	case "SHA1":
		alg = "SHA-1"
	}
	return alg, id.Hex, true
}

func spdxChecksum(integrity string) string {
	id, err := contentid.ParseSRI(integrity)
	if err != nil {
		return ""
	}
	return strings.ToUpper(id.Algo) + ": " + strings.ToUpper(id.Hex)
}

func compileRedactPattern(pattern string) (*regexp.Regexp, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, apperr.Wrap(apperr.Usage, "sbom.redact", pattern, err)
	}
	return re, nil
}

func shouldRedact(name string, redactInternal bool, pattern *regexp.Regexp) bool {
	if redactInternal && strings.HasPrefix(name, "@") {
		return true
	}
	if pattern != nil && pattern.MatchString(name) {
		return true
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
