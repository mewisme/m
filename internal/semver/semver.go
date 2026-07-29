// Package semver wraps npm-compatible range satisfaction (Masterminds/v3).
package semver

import (
	"fmt"
	"strings"

	msemver "github.com/Masterminds/semver/v3"
)

// Satisfies reports whether version matches an npm-style range.
// Dist-tags are not handled here; callers resolve tags before invoking.
func Satisfies(version, spec string) (bool, error) {
	v, err := msemver.NewVersion(stripBuild(version))
	if err != nil {
		return false, fmt.Errorf("semver: invalid version %q: %w", version, err)
	}
	c, err := msemver.NewConstraint(normalizeSpec(spec))
	if err != nil {
		return false, fmt.Errorf("semver: invalid range %q: %w", spec, err)
	}
	return c.Check(v), nil
}

// MaxSatisfying picks the highest semver-compatible version from candidates.
// Returns ("", err) when no candidate satisfies the range.
func MaxSatisfying(versions []string, spec string) (string, error) {
	c, err := msemver.NewConstraint(normalizeSpec(spec))
	if err != nil {
		return "", fmt.Errorf("semver: invalid range %q: %w", spec, err)
	}
	var best *msemver.Version
	var bestRaw string
	for _, raw := range versions {
		v, err := msemver.NewVersion(stripBuild(raw))
		if err != nil {
			continue
		}
		if !c.Check(v) {
			continue
		}
		if best == nil || v.GreaterThan(best) {
			best = v
			bestRaw = raw
		}
	}
	if best == nil {
		return "", fmt.Errorf("semver: no version satisfies %q", spec)
	}
	return bestRaw, nil
}

// Compare returns -1, 0, or 1 comparing a to b (semver order).
func Compare(a, b string) (int, error) {
	va, err := msemver.NewVersion(stripBuild(a))
	if err != nil {
		return 0, fmt.Errorf("semver: invalid version %q: %w", a, err)
	}
	vb, err := msemver.NewVersion(stripBuild(b))
	if err != nil {
		return 0, fmt.Errorf("semver: invalid version %q: %w", b, err)
	}
	return va.Compare(vb), nil
}

func stripBuild(v string) string {
	if i := strings.IndexByte(v, '('); i >= 0 {
		v = v[:i]
	}
	if i := strings.IndexByte(v, '+'); i >= 0 {
		return v[:i]
	}
	return v
}

func normalizeSpec(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "*"
	}
	return spec
}
