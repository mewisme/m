package manifest

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/mewisme/m/internal/apperr"
)

// npm package name: optional @scope/ then name; no uppercase in modern npm — allow mixed for legacy.
var nameRe = regexp.MustCompile(`^(?:@[a-z0-9~-][a-z0-9._~-]*/)?[a-z0-9~-][a-z0-9._~-]*$`)

// Validate checks name, version, and bin shapes when present.
func (d *Document) Validate() error {
	if d == nil {
		return apperr.New(apperr.Manifest, "manifest.validate", "package.json", "nil document")
	}
	if d.Name != "" {
		if err := ValidateName(d.Name); err != nil {
			return err
		}
	}
	if d.Version != "" {
		if err := ValidateVersion(d.Version); err != nil {
			return err
		}
	}
	if len(d.Bin) > 0 {
		if err := ValidateBin(d.Bin); err != nil {
			return err
		}
	}
	return nil
}

// ValidateName checks an npm package name.
func ValidateName(name string) error {
	if name == "" || strings.TrimSpace(name) != name {
		return apperr.New(apperr.Manifest, "manifest.validate", "name", "invalid package name")
	}
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		return apperr.New(apperr.Manifest, "manifest.validate", "name", "name must not start with . or _")
	}
	if len(name) > 214 {
		return apperr.New(apperr.Manifest, "manifest.validate", "name", "name too long")
	}
	if strings.ContainsAny(name, `~'!()*`) {
		return apperr.New(apperr.Manifest, "manifest.validate", "name", "name contains invalid characters")
	}
	lower := strings.ToLower(name)
	if lower != name {
		// npm forbids uppercase; still accept? Plan: reject for actionable errors.
		return apperr.New(apperr.Manifest, "manifest.validate", "name", "name must be lowercase")
	}
	if !nameRe.MatchString(name) {
		return apperr.New(apperr.Manifest, "manifest.validate", "name", fmt.Sprintf("invalid package name %q", name))
	}
	return nil
}

// ValidateVersion rejects empty/whitespace; requires a non-space token (semver-ish).
func ValidateVersion(v string) error {
	if v == "" || strings.TrimSpace(v) != v {
		return apperr.New(apperr.Manifest, "manifest.validate", "version", "invalid version")
	}
	for _, r := range v {
		if unicode.IsSpace(r) {
			return apperr.New(apperr.Manifest, "manifest.validate", "version", "version must not contain whitespace")
		}
	}
	return nil
}

// ValidateBin accepts a string path or object of name→path strings.
func ValidateBin(raw json.RawMessage) error {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if strings.TrimSpace(s) == "" {
			return apperr.New(apperr.Manifest, "manifest.validate", "bin", "empty bin path")
		}
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err == nil {
		if len(m) == 0 {
			return apperr.New(apperr.Manifest, "manifest.validate", "bin", "empty bin object")
		}
		for k, p := range m {
			if k == "" || strings.TrimSpace(p) == "" {
				return apperr.New(apperr.Manifest, "manifest.validate", "bin", "invalid bin entry")
			}
		}
		return nil
	}
	return apperr.New(apperr.Manifest, "manifest.validate", "bin", "bin must be string or object")
}
