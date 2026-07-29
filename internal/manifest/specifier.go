package manifest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// Protocol identifies how a dependency specifier resolves.
type Protocol string

const (
	ProtocolRegistry  Protocol = ""
	ProtocolNpm       Protocol = "npm"
	ProtocolWorkspace Protocol = "workspace"
	ProtocolFile      Protocol = "file"
	ProtocolLink      Protocol = "link"
	ProtocolPortal    Protocol = "portal"
	ProtocolCatalog   Protocol = "catalog"
	ProtocolGit       Protocol = "git"
	ProtocolTarball   Protocol = "tarball"
)

// Specifier is a parsed package.json dependency specifier.
type Specifier struct {
	DisplayName string   // package.json key or alias label
	TargetName  string   // resolved package name
	Range       string   // version range or local path
	Protocol    Protocol // resolution protocol
}

// ParseSpecifier parses displayName (map key) and spec (map value).
func ParseSpecifier(displayName, spec string) (Specifier, error) {
	displayName = strings.TrimSpace(displayName)
	spec = strings.TrimSpace(spec)
	if displayName == "" {
		return Specifier{}, apperr.New(apperr.Manifest, "manifest.specifier", "name", "empty dependency name")
	}
	if spec == "" {
		return Specifier{}, apperr.New(apperr.Manifest, "manifest.specifier", displayName, "empty specifier")
	}

	switch {
	case strings.HasPrefix(spec, "catalog:"):
		entry := spec[len("catalog:"):]
		if entry == "" || entry == "default" {
			return Specifier{
				DisplayName: displayName,
				TargetName:  displayName,
				Range:       displayName,
				Protocol:    ProtocolCatalog,
			}, nil
		}
		return Specifier{
			DisplayName: displayName,
			TargetName:  displayName,
			Range:       entry,
			Protocol:    ProtocolCatalog,
		}, nil
	case strings.HasPrefix(spec, "workspace:"):
		rng := spec[len("workspace:"):]
		if rng != "*" && rng != "^" && rng != "~" {
			return Specifier{}, apperr.New(apperr.Manifest, "manifest.specifier", displayName,
				fmt.Sprintf("unsupported workspace range %q", rng))
		}
		return Specifier{
			DisplayName: displayName,
			TargetName:  displayName,
			Range:       rng,
			Protocol:    ProtocolWorkspace,
		}, nil
	case strings.HasPrefix(spec, "file:"):
		return Specifier{
			DisplayName: displayName,
			TargetName:  displayName,
			Range:       spec[len("file:"):],
			Protocol:    ProtocolFile,
		}, nil
	case strings.HasPrefix(spec, "link:"):
		return Specifier{
			DisplayName: displayName,
			TargetName:  displayName,
			Range:       spec[len("link:"):],
			Protocol:    ProtocolLink,
		}, nil
	case strings.HasPrefix(spec, "portal:"):
		return Specifier{
			DisplayName: displayName,
			TargetName:  displayName,
			Range:       spec[len("portal:"):],
			Protocol:    ProtocolPortal,
		}, nil
	case strings.HasPrefix(spec, "tarball:"):
		rng := strings.TrimSpace(spec[len("tarball:"):])
		if rng == "" {
			return Specifier{}, apperr.New(apperr.Manifest, "manifest.specifier", displayName, "empty tarball path")
		}
		return Specifier{
			DisplayName: displayName,
			TargetName:  displayName,
			Range:       rng,
			Protocol:    ProtocolTarball,
		}, nil
	case isGitSpecifier(spec):
		return parseGitSpecifier(displayName, spec)
	case strings.HasPrefix(spec, "npm:"):
		return parseNpmSpecifier(displayName, spec)
	default:
		return Specifier{
			DisplayName: displayName,
			TargetName:  displayName,
			Range:       spec,
			Protocol:    ProtocolRegistry,
		}, nil
	}
}

func isGitSpecifier(spec string) bool {
	return strings.HasPrefix(spec, "git+https:") ||
		strings.HasPrefix(spec, "git+ssh:") ||
		strings.HasPrefix(spec, "git+file:") ||
		strings.HasPrefix(spec, "git:") ||
		strings.HasPrefix(spec, "github:")
}

func parseGitSpecifier(displayName, spec string) (Specifier, error) {
	rng, err := normalizeGitSpecifierRange(spec)
	if err != nil {
		return Specifier{}, apperr.Wrap(apperr.Manifest, "manifest.specifier", displayName, err)
	}
	return Specifier{
		DisplayName: displayName,
		TargetName:  displayName,
		Range:       rng,
		Protocol:    ProtocolGit,
	}, nil
}

func normalizeGitSpecifierRange(spec string) (string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", fmt.Errorf("empty git specifier")
	}
	if strings.HasPrefix(spec, "github:") {
		rest := strings.TrimSpace(spec[len("github:"):])
		if rest == "" {
			return "", fmt.Errorf("empty github: repository")
		}
		ref := ""
		if i := strings.IndexByte(rest, '#'); i >= 0 {
			ref = strings.TrimSpace(rest[i+1:])
			rest = strings.TrimSpace(rest[:i])
		}
		if rest == "" {
			return "", fmt.Errorf("empty github: repository")
		}
		url := "https://github.com/" + strings.TrimSuffix(rest, ".git") + ".git"
		if ref != "" {
			url += "#" + ref
		}
		return url, nil
	}
	for _, prefix := range []string{"git+https:", "git+ssh:", "git+file:", "git:"} {
		if strings.HasPrefix(spec, prefix) {
			rng := strings.TrimSpace(spec[len(prefix):])
			if rng == "" {
				return "", fmt.Errorf("empty git url")
			}
			return rng, nil
		}
	}
	return "", fmt.Errorf("unsupported git specifier %q", spec)
}

func parseNpmSpecifier(displayName, spec string) (Specifier, error) {
	rest := spec[len("npm:"):]
	if i := strings.Index(rest, "@npm:"); i >= 0 {
		alias := rest[:i]
		tail := rest[i+len("@npm:"):]
		target, rng, err := splitNameRange(tail)
		if err != nil {
			return Specifier{}, apperr.Wrap(apperr.Manifest, "manifest.specifier", displayName, err)
		}
		disp := alias
		if disp == "" {
			disp = displayName
		}
		return Specifier{
			DisplayName: disp,
			TargetName:  target,
			Range:       rng,
			Protocol:    ProtocolNpm,
		}, nil
	}
	target, rng, err := splitNameRange(rest)
	if err != nil {
		return Specifier{}, apperr.Wrap(apperr.Manifest, "manifest.specifier", displayName, err)
	}
	return Specifier{
		DisplayName: displayName,
		TargetName:  target,
		Range:       rng,
		Protocol:    ProtocolNpm,
	}, nil
}

func splitNameRange(s string) (name, verRange string, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", fmt.Errorf("empty name@range")
	}
	if strings.HasPrefix(s, "@") {
		slash := strings.IndexByte(s, '/')
		if slash < 0 {
			return "", "", fmt.Errorf("invalid scoped name %q", s)
		}
		at := strings.Index(s[slash:], "@")
		if at < 0 {
			return s, "*", nil
		}
		return s[:slash+at], s[slash+at+1:], nil
	}
	at := strings.LastIndexByte(s, '@')
	if at < 0 {
		return s, "*", nil
	}
	return s[:at], s[at+1:], nil
}

// FlattenOverrides converts nested overrides objects into dotted paths.
func FlattenOverrides(raw map[string]json.RawMessage) (map[string]string, error) {
	if raw == nil {
		return nil, nil
	}
	out := make(map[string]string, len(raw))
	var walk func(prefix string, v json.RawMessage) error
	walk = func(prefix string, v json.RawMessage) error {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			if prefix == "" {
				return apperr.New(apperr.Manifest, "manifest.overrides", "", "empty override path")
			}
			out[prefix] = s
			return nil
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(v, &m); err != nil {
			return apperr.Wrap(apperr.Manifest, "manifest.overrides", prefix, err)
		}
		if len(m) == 0 {
			return apperr.New(apperr.Manifest, "manifest.overrides", prefix, "empty override object")
		}
		for k, child := range m {
			key := k
			if prefix != "" {
				if k == "." {
					key = prefix
				} else {
					key = prefix + "." + k
				}
			}
			if err := walk(key, child); err != nil {
				return err
			}
		}
		return nil
	}
	for k, v := range raw {
		if err := walk(k, v); err != nil {
			return nil, err
		}
	}
	return out, nil
}
