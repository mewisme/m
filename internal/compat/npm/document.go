package npm

import (
	"encoding/json"

	"github.com/mewisme/mew/internal/lockfile"
)

const (
	FormatV2 = "npm-v2"
	FormatV3 = "npm-v3"
)

// Document is the parsed npm package-lock / shrinkwrap JSON model.
type Document struct {
	LockfileVersion int                     `json:"lockfileVersion"`
	Name            string                  `json:"name,omitempty"`
	Requires        bool                    `json:"requires,omitempty"`
	Packages        map[string]PackageEntry `json:"packages,omitempty"`
	Dependencies    map[string]LegacyDep    `json:"dependencies,omitempty"`
	Extensions      lockfile.Extensions     `json:"-"`
	Detection       lockfile.Detection      `json:"-"`
}

// PackageEntry is one packages-map record in npm lock v2/v3.
type PackageEntry struct {
	Name                 string                     `json:"name,omitempty"`
	Version              string                     `json:"version,omitempty"`
	Resolved             string                     `json:"resolved,omitempty"`
	Integrity            string                     `json:"integrity,omitempty"`
	Link                 bool                       `json:"link,omitempty"`
	Dev                  bool                       `json:"dev,omitempty"`
	DevOptional          bool                       `json:"devOptional,omitempty"`
	Optional             bool                       `json:"optional,omitempty"`
	Dependencies         map[string]string          `json:"dependencies,omitempty"`
	DevDependencies      map[string]string          `json:"devDependencies,omitempty"`
	OptionalDependencies map[string]string          `json:"optionalDependencies,omitempty"`
	PeerDependencies     map[string]string          `json:"peerDependencies,omitempty"`
	BundledDependencies  []string                   `json:"bundledDependencies,omitempty"`
	Workspaces           []string                   `json:"workspaces,omitempty"`
	Extra                map[string]json.RawMessage `json:"-"`
}

// LegacyDep is the nested dependencies tree used by lockfile v1/v2.
type LegacyDep struct {
	Version   string                     `json:"version,omitempty"`
	Resolved  string                     `json:"resolved,omitempty"`
	Integrity string                     `json:"integrity,omitempty"`
	Requires  bool                       `json:"requires,omitempty"`
	Deps      map[string]LegacyDep       `json:"dependencies,omitempty"`
	Extra     map[string]json.RawMessage `json:"-"`
}

// Clone returns a deep copy of the document.
func (d *Document) Clone() *Document {
	if d == nil {
		return nil
	}
	out := *d
	out.Packages = make(map[string]PackageEntry, len(d.Packages))
	for k, v := range d.Packages {
		out.Packages[k] = v.clone()
	}
	out.Dependencies = cloneLegacyDeps(d.Dependencies)
	if len(d.Extensions) > 0 {
		out.Extensions = make(lockfile.Extensions, len(d.Extensions))
		for k, v := range d.Extensions {
			out.Extensions[k] = append(json.RawMessage(nil), v...)
		}
	}
	return &out
}

func (e PackageEntry) clone() PackageEntry {
	out := e
	out.Dependencies = cloneStringMap(e.Dependencies)
	out.DevDependencies = cloneStringMap(e.DevDependencies)
	out.OptionalDependencies = cloneStringMap(e.OptionalDependencies)
	out.PeerDependencies = cloneStringMap(e.PeerDependencies)
	if len(e.BundledDependencies) > 0 {
		out.BundledDependencies = append([]string(nil), e.BundledDependencies...)
	}
	if len(e.Workspaces) > 0 {
		out.Workspaces = append([]string(nil), e.Workspaces...)
	}
	if len(e.Extra) > 0 {
		out.Extra = make(map[string]json.RawMessage, len(e.Extra))
		for k, v := range e.Extra {
			out.Extra[k] = append(json.RawMessage(nil), v...)
		}
	}
	return out
}

func cloneStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func cloneLegacyDeps(m map[string]LegacyDep) map[string]LegacyDep {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]LegacyDep, len(m))
	for k, v := range m {
		child := v
		child.Deps = cloneLegacyDeps(v.Deps)
		if len(v.Extra) > 0 {
			child.Extra = make(map[string]json.RawMessage, len(v.Extra))
			for ek, ev := range v.Extra {
				child.Extra[ek] = append(json.RawMessage(nil), ev...)
			}
		}
		out[k] = child
	}
	return out
}
