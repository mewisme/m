package registry

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mewisme/m/internal/apperr"
)

// ParsePackument validates and normalizes packument JSON.
func ParsePackument(data []byte) (*Packument, error) {
	if len(data) == 0 {
		return nil, apperr.New(apperr.Network, "registry.parse", "packument", "empty body")
	}
	var p Packument
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, apperr.Wrap(apperr.Network, "registry.parse", "packument", err)
	}
	if p.Name == "" {
		return nil, apperr.New(apperr.Network, "registry.parse", "packument", "missing name")
	}
	if p.Versions == nil {
		p.Versions = map[string]VersionMeta{}
	}
	if p.DistTags == nil {
		p.DistTags = map[string]string{}
	}
	for ver, meta := range p.Versions {
		if meta.Version == "" {
			meta.Version = ver
		}
		if meta.Dist.Tarball == "" && meta.Dist.Integrity == "" {
			return nil, apperr.New(apperr.Network, "registry.parse", p.Name+"@"+ver,
				"version missing dist.tarball and dist.integrity")
		}
		p.Versions[ver] = meta
	}
	p.Raw = append(json.RawMessage(nil), data...)
	return &p, nil
}

// SelectVersion resolves a version or dist-tag to VersionMeta.
func (p *Packument) SelectVersion(spec string) (*VersionMeta, error) {
	if p == nil {
		return nil, apperr.New(apperr.Network, "registry.select", "", "nil packument")
	}
	if spec == "" || spec == "latest" {
		if tag, ok := p.DistTags["latest"]; ok {
			spec = tag
		}
	}
	if v, ok := p.DistTags[spec]; ok {
		spec = v
	}
	meta, ok := p.Versions[spec]
	if !ok {
		return nil, apperr.New(apperr.NotFound, "registry.select", p.Name+"@"+spec,
			fmt.Sprintf("version %q not in packument", spec))
	}
	return &meta, nil
}

// SortedVersions returns version keys sorted.
func (p *Packument) SortedVersions() []string {
	keys := make([]string, 0, len(p.Versions))
	for k := range p.Versions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// EncodeNamePath encodes a package name for the registry URL path.
func EncodeNamePath(name string) string {
	if strings.HasPrefix(name, "@") {
		if i := strings.IndexByte(name, '/'); i > 0 {
			return name[:i] + "%2F" + name[i+1:]
		}
	}
	return name
}
