package resolver

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/manifest"
)

// GitExtensionKey is the m.lock extensions key for git-sourced packages.
const GitExtensionKey = "mew.resolver/git"

var gitCommitPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// GitSource records a resolved git dependency in the lock graph.
type GitSource struct {
	URL       string `json:"url"`
	Commit    string `json:"commit"`
	Integrity string `json:"integrity,omitempty"`
}

// ParsedGitSpec is a normalized git dependency URL and optional ref.
type ParsedGitSpec struct {
	URL string
	Ref string
}

// ParseGitRange parses a git specifier range (post-protocol URL#ref).
func ParseGitRange(rng string) (ParsedGitSpec, error) {
	rng = strings.TrimSpace(rng)
	if rng == "" {
		return ParsedGitSpec{}, fmt.Errorf("empty git url")
	}
	ref := ""
	if i := strings.IndexByte(rng, '#'); i >= 0 {
		ref = strings.TrimSpace(rng[i+1:])
		rng = strings.TrimSpace(rng[:i])
	}
	if rng == "" {
		return ParsedGitSpec{}, fmt.Errorf("empty git url")
	}
	normalized, err := NormalizeGitURL(rng)
	if err != nil {
		return ParsedGitSpec{}, err
	}
	if err := ValidateGitURL(normalized); err != nil {
		return ParsedGitSpec{}, err
	}
	if ref != "" && !isGitRef(ref) {
		return ParsedGitSpec{}, fmt.Errorf("invalid git ref %q", ref)
	}
	return ParsedGitSpec{URL: normalized, Ref: ref}, nil
}

// NormalizeGitURL canonicalizes supported git URL forms.
func NormalizeGitURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty git url")
	}
	if strings.HasPrefix(raw, "git@") {
		if i := strings.IndexByte(raw, ':'); i > 0 {
			host := raw[4:i]
			path := strings.TrimPrefix(raw[i+1:], "/")
			raw = "ssh://git@" + host + "/" + path
		}
	}
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid git url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "https", "http", "ssh", "git", "file":
	default:
		return "", fmt.Errorf("unsupported git scheme %q", u.Scheme)
	}
	if u.Scheme != "file" && strings.TrimSpace(u.Host) == "" {
		return "", fmt.Errorf("git url missing host")
	}
	if u.User != nil && (u.User.Username() == "" || strings.Contains(u.User.Username(), "@")) {
		return "", fmt.Errorf("invalid git url user")
	}
	u.Fragment = ""
	u.RawQuery = ""
	out := u.String()
	if strings.HasSuffix(out, "/") {
		out = strings.TrimSuffix(out, "/")
	}
	return out, nil
}

// ValidateGitURL rejects unsafe git URL forms before network I/O.
func ValidateGitURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid git url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "https", "http", "ssh", "git", "file":
	default:
		return fmt.Errorf("unsupported git scheme %q", u.Scheme)
	}
	if u.Scheme != "file" {
		if strings.TrimSpace(u.Host) == "" {
			return fmt.Errorf("git url missing host")
		}
	}
	path := strings.TrimSpace(u.Path)
	if u.Scheme != "file" && path == "" {
		return fmt.Errorf("git url missing repository path")
	}
	if strings.Contains(raw, "..") && u.Scheme == "file" {
		return fmt.Errorf("git file url must not contain ..")
	}
	return nil
}

func isGitRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	if gitCommitPattern.MatchString(strings.ToLower(ref)) {
		return true
	}
	return isGitBranchish(ref)
}

func isGitBranchish(ref string) bool {
	if ref == "" || len(ref) > 256 {
		return false
	}
	for _, r := range ref {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.', r == '_', r == '/', r == '-':
		default:
			return false
		}
	}
	return true
}

func (s *resolveState) processGit(item workItem) error {
	if item.depth > maxDepth {
		return apperr.New(apperr.Resolve, "resolver.limit", item.name,
			fmt.Sprintf("resolution depth exceeded %d", maxDepth))
	}
	parsed, err := ParseGitRange(resolveGitRange(s.proj.Root, item.declarerPath, item.rng))
	if err != nil {
		return apperr.Wrap(apperr.Resolve, "resolver.git", item.name, err)
	}
	commit, err := s.resolveGitCommit(item, parsed)
	if err != nil {
		return err
	}
	targetDir, err := gitPeekPackageDir(s.ctx, parsed.URL, commit, s.pol != nil && s.pol.Offline)
	if err != nil {
		return apperr.Wrap(apperr.Resolve, "resolver.git", item.name, err)
	}
	defer func() { _ = os.RemoveAll(targetDir) }()
	_, version, err := readLocalPackage(targetDir, item.name, manifest.ProtocolGit)
	if err != nil {
		return apperr.Wrap(apperr.Resolve, "resolver.git", item.name, err)
	}
	id, key := s.packageKeyForInstance(item, version, nil)

	decision := ResolutionDecision{
		Package:   item.name,
		Requested: item.spec,
		Selected:  version,
		Reason:    "git",
	}
	s.decisions = append(s.decisions, decision)

	edgeRange := item.spec
	if edgeRange == "" {
		edgeRange = "git+" + parsed.URL
		if parsed.Ref != "" {
			edgeRange += "#" + parsed.Ref
		}
	}
	s.b.EdgeEx(item.from, item.display, key, item.kind, edgeRange, false)
	s.recordProvides(item.from, item.display, key)

	if _, ok := s.seenPkg[key]; ok {
		return nil
	}
	if _, ok := s.resolving[basePackageKey(item.name, version)]; ok {
		return nil
	}
	if len(s.seenPkg) >= maxPackages {
		return apperr.New(apperr.Resolve, "resolver.limit", item.name,
			fmt.Sprintf("resolution package count exceeded %d", maxPackages))
	}
	s.resolving[basePackageKey(item.name, version)] = struct{}{}
	defer delete(s.resolving, basePackageKey(item.name, version))

	s.seenPkg[key] = struct{}{}
	s.b.Package(id, "", "")
	s.gitSources[key] = GitSource{URL: parsed.URL, Commit: commit}
	s.pkgEnv[key] = append([]string(nil), item.envKeys...)
	s.pkgFrom[key] = item.from

	return s.expandLocalManifestAbs(key, targetDir, item.depth, item.path, item.envKeys)
}

func (s *resolveState) resolveGitCommit(item workItem, parsed ParsedGitSpec) (string, error) {
	if parsed.Ref != "" && gitCommitPattern.MatchString(strings.ToLower(parsed.Ref)) {
		return strings.ToLower(parsed.Ref), nil
	}
	if hv := s.hints.version(item.name, item.spec); hv != "" && gitCommitPattern.MatchString(strings.ToLower(hv)) {
		return strings.ToLower(hv), nil
	}
	offline := s.pol != nil && s.pol.Offline
	return ResolveGitCommit(s.ctx, parsed.URL, parsed.Ref, offline)
}

func targetDirToRel(root, abs string) string {
	rel, err := relPathToRoot(root, abs)
	if err != nil {
		return abs
	}
	return rel
}

// HasGitSources reports whether extensions contain git source metadata.
func HasGitSources(ext lockfile.Extensions) bool {
	sources, err := DecodeGitSources(ext)
	return err == nil && len(sources) > 0
}

// DecodeGitSources parses the mew.resolver/git extension payload.
func DecodeGitSources(ext lockfile.Extensions) (map[string]GitSource, error) {
	if len(ext) == 0 {
		return nil, nil
	}
	raw, ok := ext[GitExtensionKey]
	if !ok {
		return nil, nil
	}
	var sources map[string]GitSource
	if err := json.Unmarshal(raw, &sources); err != nil {
		return nil, err
	}
	return sources, nil
}

func resolveGitRange(root, declarerPath, rng string) string {
	rng = strings.TrimSpace(rng)
	ref := ""
	if i := strings.IndexByte(rng, '#'); i >= 0 {
		ref = rng[i:]
		rng = strings.TrimSpace(rng[:i])
	}
	if strings.Contains(rng, "://") || strings.HasPrefix(rng, "git@") {
		return rng + ref
	}
	base := root
	if declarerPath != "" && declarerPath != "." {
		base = filepath.Join(root, filepath.FromSlash(declarerPath))
	}
	abs := filepath.Clean(filepath.Join(base, filepath.FromSlash(rng)))
	abs, err := filepath.Abs(abs)
	if err != nil {
		return rng + ref
	}
	return fileURLForPath(abs) + ref
}

func fileURLForPath(abs string) string {
	slash := filepath.ToSlash(abs)
	if runtime.GOOS == "windows" {
		if len(slash) >= 2 && slash[1] == ':' {
			return "file:///" + slash
		}
		return "file:///" + slash
	}
	return "file://" + slash
}
