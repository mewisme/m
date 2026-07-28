package pnpm

import (
	"github.com/mewisme/mew/internal/lockfile"
)

// Generation constants for Detection.Format.
const (
	FormatV6  = "pnpm-v6"
	FormatV9  = "pnpm-v9"
	FormatV10 = "pnpm-v10"
	FormatV11 = "pnpm-v11"
)

// Document is a parsed pnpm lock with extension preservation.
type Document struct {
	LockfileVersion string
	Settings        map[string]any
	Importers       map[string]ImporterSection
	Packages        map[string]PackageEntry
	Snapshots       map[string]map[string]any
	Dependencies    map[string]ImporterDep // v6 root dependencies
	Extensions      lockfile.Extensions
	Detection       lockfile.Detection
}

// ImporterSection mirrors pnpm importer blocks.
type ImporterSection struct {
	Dependencies    map[string]ImporterDep `yaml:"dependencies,omitempty"`
	DevDependencies map[string]ImporterDep `yaml:"devDependencies,omitempty"`
}

// ImporterDep is a single importer dependency entry.
type ImporterDep struct {
	Specifier string `yaml:"specifier"`
	Version   string `yaml:"version"`
}

// PackageEntry holds a packages map value with generation-specific fields.
type PackageEntry struct {
	Resolution   map[string]any    `yaml:"resolution,omitempty"`
	Engines      map[string]any    `yaml:"engines,omitempty"`
	Dependencies map[string]string `yaml:"dependencies,omitempty"`
	Checksum     string            `yaml:"checksum,omitempty"`
	BuildPolicy  any               `yaml:"buildPolicy,omitempty"`
	Extra        map[string]any    `yaml:",inline"`
}

// Clone returns a deep copy of the document for encode.
func (d *Document) Clone() *Document {
	if d == nil {
		return nil
	}
	out := *d
	out.Settings = cloneMap(d.Settings)
	out.Importers = cloneImporters(d.Importers)
	out.Packages = clonePackages(d.Packages)
	out.Snapshots = cloneSnapshots(d.Snapshots)
	out.Dependencies = cloneDeps(d.Dependencies)
	out.Extensions = cloneExtensions(d.Extensions)
	return &out
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func cloneImporters(in map[string]ImporterSection) map[string]ImporterSection {
	if in == nil {
		return nil
	}
	out := make(map[string]ImporterSection, len(in))
	for k, v := range in {
		sec := ImporterSection{}
		sec.Dependencies = cloneDeps(v.Dependencies)
		sec.DevDependencies = cloneDeps(v.DevDependencies)
		out[k] = sec
	}
	return out
}

func cloneDeps(in map[string]ImporterDep) map[string]ImporterDep {
	if in == nil {
		return nil
	}
	out := make(map[string]ImporterDep, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func clonePackages(in map[string]PackageEntry) map[string]PackageEntry {
	if in == nil {
		return nil
	}
	out := make(map[string]PackageEntry, len(in))
	for k, v := range in {
		e := v
		e.Resolution = cloneMap(v.Resolution)
		e.Engines = cloneMap(v.Engines)
		e.Extra = cloneMap(v.Extra)
		if v.Dependencies != nil {
			e.Dependencies = make(map[string]string, len(v.Dependencies))
			for dk, dv := range v.Dependencies {
				e.Dependencies[dk] = dv
			}
		}
		out[k] = e
	}
	return out
}

func cloneSnapshots(in map[string]map[string]any) map[string]map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneMap(v)
	}
	return out
}

func cloneExtensions(in lockfile.Extensions) lockfile.Extensions {
	if in == nil {
		return nil
	}
	out := make(lockfile.Extensions, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
