package dlx

import (
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// PackageSpec is a normalized registry package reference.
type PackageSpec struct {
	Name    string
	Version string // empty means latest; explicit version, range, or dist-tag
	Raw     string
}

// ParsePackageSpec parses registry-only package specs for mx.
func ParsePackageSpec(spec string) (PackageSpec, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return PackageSpec{}, apperr.New(apperr.Usage, "dlx.spec", "", "package spec required")
	}
	lower := strings.ToLower(spec)
	for _, prefix := range []string{"file:", "link:", "portal:", "workspace:", "git:", "git+ssh:", "http:", "https:", "npm:"} {
		if strings.HasPrefix(lower, prefix) {
			return PackageSpec{}, apperr.New(apperr.Unsupported, "dlx.spec", spec, "unsupported package protocol")
		}
	}
	if strings.Contains(spec, "://") {
		return PackageSpec{}, apperr.New(apperr.Unsupported, "dlx.spec", spec, "unsupported package protocol")
	}
	name, version := splitRegistrySpec(spec)
	if name == "" {
		return PackageSpec{}, apperr.New(apperr.Usage, "dlx.spec", spec, "invalid package spec")
	}
	if strings.Contains(name, " ") {
		return PackageSpec{}, apperr.New(apperr.Usage, "dlx.spec", spec, "invalid package name")
	}
	return PackageSpec{Name: name, Version: version, Raw: spec}, nil
}

func splitRegistrySpec(spec string) (name, version string) {
	if !strings.HasPrefix(spec, "@") {
		if i := strings.LastIndex(spec, "@"); i > 0 {
			return spec[:i], spec[i+1:]
		}
		return spec, "latest"
	}
	slash := strings.Index(spec, "/")
	if slash < 0 {
		return spec, "latest"
	}
	rest := spec[slash+1:]
	if i := strings.Index(rest, "@"); i >= 0 {
		return spec[:slash+1+i], rest[i+1:]
	}
	return spec, "latest"
}

// HasExplicitVersion reports whether the spec pins a version, range, or tag.
func (s PackageSpec) HasExplicitVersion() bool {
	return strings.TrimSpace(s.Version) != "" && s.Version != "latest"
}

// UnscopedName returns the final segment of a scoped package name.
func UnscopedName(name string) string {
	if strings.HasPrefix(name, "@") {
		if i := strings.LastIndex(name, "/"); i >= 0 {
			return name[i+1:]
		}
	}
	return name
}
