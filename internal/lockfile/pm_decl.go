package lockfile

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// PM evidence states for packageManager / devEngines parsing.
const (
	PMEvidenceCertain     = "certain"
	PMEvidenceNone        = "none"
	PMEvidenceInvalid     = "invalid"
	PMEvidenceUnsupported = "unsupported"
)

// PMDeclaration is a parsed packageManager-style field value.
type PMDeclaration struct {
	Name          string
	ExactVersion  string
	ProducerMajor int
	EvidenceState string
}

// ParsePMDeclaration parses a packageManager or devEngines.packageManager value.
func ParsePMDeclaration(field, value string) (PMDeclaration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return PMDeclaration{EvidenceState: PMEvidenceNone}, nil
	}
	name := value
	if i := strings.IndexByte(value, '@'); i >= 0 {
		name = value[:i]
	}
	if strings.ToLower(strings.TrimSpace(name)) != "pnpm" {
		return PMDeclaration{}, fmt.Errorf("%s: expected pnpm@<version>, got %q", field, value)
	}
	at := strings.LastIndexByte(value, '@')
	if at < 0 || at >= len(value)-1 {
		return PMDeclaration{Name: "pnpm", EvidenceState: PMEvidenceNone}, nil
	}
	ver := strings.TrimSpace(value[at+1:])
	if ver == "" {
		return PMDeclaration{Name: "pnpm", EvidenceState: PMEvidenceNone}, nil
	}
	if isPMVersionRangeOrTag(ver) {
		return PMDeclaration{Name: "pnpm", ExactVersion: ver, EvidenceState: PMEvidenceInvalid},
			fmt.Errorf("%s: version ranges and tags are not supported in %q", field, value)
	}
	major, exact, err := pmMajorFromVersion(ver)
	if err != nil {
		return PMDeclaration{Name: "pnpm", ExactVersion: ver, EvidenceState: PMEvidenceInvalid}, err
	}
	if major != 0 && (major < 9 || major > 11) {
		return PMDeclaration{Name: "pnpm", ExactVersion: exact, ProducerMajor: major, EvidenceState: PMEvidenceUnsupported},
			fmt.Errorf("%s: pnpm@%d is unsupported; regenerate with pnpm 9, 10, or 11", field, major)
	}
	if major == 0 {
		return PMDeclaration{Name: "pnpm", ExactVersion: exact, EvidenceState: PMEvidenceNone}, nil
	}
	return PMDeclaration{
		Name:          "pnpm",
		ExactVersion:  exact,
		ProducerMajor: major,
		EvidenceState: PMEvidenceCertain,
	}, nil
}

func isPMVersionRangeOrTag(ver string) bool {
	ver = strings.TrimSpace(ver)
	if ver == "" {
		return false
	}
	if strings.ContainsAny(ver, "^~>=< *xX") {
		return true
	}
	switch strings.ToLower(ver) {
	case "latest", "next", "beta", "rc", "canary":
		return true
	}
	return false
}

func pmMajorFromVersion(ver string) (major int, exact string, err error) {
	exact = strings.TrimSpace(ver)
	parsed, parseErr := semver.NewVersion(exact)
	if parseErr != nil {
		return 0, exact, fmt.Errorf("unrecognized pnpm version %q: %v", ver, parseErr)
	}
	major64 := parsed.Major()
	if major64 > 11 {
		// still valid semver; unsupported major handled by caller
	}
	return int(major64), parsed.String(), nil
}
