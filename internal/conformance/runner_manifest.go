package conformance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
)

// RunnerSuite describes one runner certification suite.
type RunnerSuite struct {
	ID            string            `json:"id"`
	Group         string            `json:"group"`
	Package       string            `json:"package"`
	Run           string            `json:"run"`
	ExpectedTests []string          `json:"expectedTests"`
	Timeout       string            `json:"timeout"`
	Required      bool              `json:"required"`
	Probe         bool              `json:"probe"`
	Platforms     []string          `json:"platforms"`
	Isolation     string            `json:"isolation"`
	NetworkPolicy string            `json:"networkPolicy"`
	Environment   map[string]string `json:"environment"`
	Description   string            `json:"description"`
	WaiverPolicy  string            `json:"waiverPolicy"`
	WaiverIDs     []string          `json:"waiverIds"`
}

// RunnerManifest is the runner certification matrix definition.
type RunnerManifest struct {
	SchemaVersion int           `json:"schemaVersion"`
	Matrix        string        `json:"matrix"`
	Suites        []RunnerSuite `json:"suites"`
}

// RunnerManifestPath returns tests/conformance/runner-matrix/manifest.json.
func RunnerManifestPath(repoRoot string) string {
	return filepath.Join(repoRoot, "tests", "conformance", "runner-matrix", "manifest.json")
}

// RunnerWaiverPath returns tests/conformance/runner-matrix/waivers.v1.json.
func RunnerWaiverPath(repoRoot string) string {
	return filepath.Join(repoRoot, "tests", "conformance", "runner-matrix", "waivers.v1.json")
}

// LoadRunnerManifest reads and validates the runner matrix manifest.
func LoadRunnerManifest(path string) (RunnerManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RunnerManifest{}, apperr.Wrap(apperr.NotFound, "conformance.runner.manifest", path, err)
		}
		return RunnerManifest{}, apperr.Wrap(apperr.IO, "conformance.runner.manifest", path, err)
	}
	var m RunnerManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return RunnerManifest{}, apperr.Wrap(apperr.Manifest, "conformance.runner.manifest", path, err)
	}
	if err := validateRunnerManifest(&m); err != nil {
		return RunnerManifest{}, err
	}
	return m, nil
}

func validateRunnerManifest(m *RunnerManifest) error {
	if m.SchemaVersion != RunnerManifestSchema {
		return apperr.New(apperr.Unsupported, "conformance.runner.manifest", "", fmt.Sprintf("unsupported schema %d", m.SchemaVersion))
	}
	if m.Matrix != RunnerMatrix {
		return apperr.New(apperr.Manifest, "conformance.runner.manifest", "", fmt.Sprintf("unsupported matrix %q", m.Matrix))
	}
	if len(m.Suites) == 0 {
		return apperr.New(apperr.Manifest, "conformance.runner.manifest", "", "manifest has no suites")
	}
	if len(m.Suites) > RunnerMaxSuiteCount {
		return apperr.New(apperr.Manifest, "conformance.runner.manifest", "", "too many suites")
	}
	seen := map[string]struct{}{}
	for i := range m.Suites {
		if err := validateRunnerSuite(&m.Suites[i]); err != nil {
			return err
		}
		if _, ok := seen[m.Suites[i].ID]; ok {
			return apperr.New(apperr.Manifest, "conformance.runner.manifest", m.Suites[i].ID, "duplicate suite id")
		}
		seen[m.Suites[i].ID] = struct{}{}
	}
	sortRunnerManifestForDigest(m)
	return nil
}

func validateRunnerSuite(s *RunnerSuite) error {
	if strings.TrimSpace(s.ID) == "" {
		return apperr.New(apperr.Manifest, "conformance.runner.suite", "", "suite missing id")
	}
	if _, ok := validRunnerGroups[s.Group]; !ok {
		return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, fmt.Sprintf("invalid group %q", s.Group))
	}
	if err := validateRunnerPackage(s.Package); err != nil {
		return apperr.Wrap(apperr.Manifest, "conformance.runner.suite", s.ID, err)
	}
	if err := validateRunRegex(s.Run); err != nil {
		return apperr.Wrap(apperr.Manifest, "conformance.runner.suite", s.ID, err)
	}
	if len(s.ExpectedTests) == 0 {
		return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, "expectedTests must be non-empty")
	}
	if len(s.ExpectedTests) > RunnerMaxExpectedTests {
		return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, "too many expectedTests")
	}
	if !sort.StringsAreSorted(s.ExpectedTests) {
		return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, "expectedTests must be sorted")
	}
	seenTests := map[string]struct{}{}
	for _, name := range s.ExpectedTests {
		if strings.TrimSpace(name) == "" {
			return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, "expectedTests contains empty name")
		}
		if _, ok := seenTests[name]; ok {
			return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, fmt.Sprintf("duplicate expected test %q", name))
		}
		seenTests[name] = struct{}{}
	}
	if _, err := time.ParseDuration(s.Timeout); err != nil {
		return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, fmt.Sprintf("invalid timeout %q", s.Timeout))
	}
	if _, ok := validIsolationPolicies[s.Isolation]; !ok {
		return apperr.New(apperr.Unsupported, "conformance.runner.suite", s.ID, fmt.Sprintf("unsupported isolation %q", s.Isolation))
	}
	if _, ok := validNetworkPolicies[s.NetworkPolicy]; !ok {
		return apperr.New(apperr.Unsupported, "conformance.runner.suite", s.ID, fmt.Sprintf("unsupported networkPolicy %q", s.NetworkPolicy))
	}
	if _, ok := validWaiverPolicies[s.WaiverPolicy]; !ok {
		return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, fmt.Sprintf("invalid waiverPolicy %q", s.WaiverPolicy))
	}
	if !sort.StringsAreSorted(s.WaiverIDs) {
		return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, "waiverIds must be sorted")
	}
	if len(s.Platforms) == 0 {
		return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, "platforms must be non-empty")
	}
	sort.Strings(s.Platforms)
	for _, p := range s.Platforms {
		if p != "linux" && p != "darwin" && p != "windows" {
			return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, fmt.Sprintf("invalid platform %q", p))
		}
	}
	if err := validateEnvironmentOverlay(s.Environment); err != nil {
		return apperr.Wrap(apperr.Manifest, "conformance.runner.suite", s.ID, err)
	}
	if strings.TrimSpace(s.Description) == "" {
		return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, "description required")
	}
	return nil
}

func validateRunRegex(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("empty run regex")
	}
	if !strings.HasPrefix(pattern, "^") || !strings.HasSuffix(pattern, "$") {
		return fmt.Errorf("run regex must be anchored with ^ and $")
	}
	if strings.Contains(pattern, `\|`) {
		return fmt.Errorf("run regex must not use escaped pipe alternation")
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("invalid run regex: %w", err)
	}
	return nil
}

func validateRunnerPackage(pkg string) error {
	if !strings.HasPrefix(pkg, "./") {
		return fmt.Errorf("package must be ./-prefixed")
	}
	if strings.Contains(pkg, "..") {
		return fmt.Errorf("package path must not contain parent segments")
	}
	clean := filepath.Clean(strings.TrimPrefix(pkg, "./"))
	if clean == "." || clean == "" {
		return fmt.Errorf("invalid package path")
	}
	return nil
}

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var harnessEnvAllowlist = map[string]struct{}{
	"MEW_CONFORMANCE_TTY":           {},
	"MEW_TEST_REGISTRY_URL":         {},
	"MEW_TEST_NETWORK_POLICY":       {},
	"CGO_ENABLED":                   {},
	"MEW_CONFORMANCE_REQUIRE_TOOLS": {},
}

func validateEnvironmentOverlay(env map[string]string) error {
	if env == nil {
		return nil
	}
	seen := map[string]struct{}{}
	for k := range env {
		if !envNamePattern.MatchString(k) {
			return fmt.Errorf("invalid environment key %q", k)
		}
		if _, ok := seen[k]; ok {
			return fmt.Errorf("duplicate environment key %q", k)
		}
		seen[k] = struct{}{}
	}
	return nil
}

func sortRunnerManifestForDigest(m *RunnerManifest) {
	sort.SliceStable(m.Suites, func(i, j int) bool {
		gi := groupIndex(m.Suites[i].Group)
		gj := groupIndex(m.Suites[j].Group)
		if gi != gj {
			return gi < gj
		}
		return m.Suites[i].ID < m.Suites[j].ID
	})
	for i := range m.Suites {
		sort.Strings(m.Suites[i].Platforms)
		sort.Strings(m.Suites[i].ExpectedTests)
		sort.Strings(m.Suites[i].WaiverIDs)
		if m.Suites[i].Environment != nil {
			keys := make([]string, 0, len(m.Suites[i].Environment))
			for k := range m.Suites[i].Environment {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			normalized := make(map[string]string, len(keys))
			for _, k := range keys {
				normalized[k] = m.Suites[i].Environment[k]
			}
			m.Suites[i].Environment = normalized
		}
	}
}

func groupIndex(group string) int {
	for i, g := range runnerGroupOrder {
		if g == group {
			return i
		}
	}
	return len(runnerGroupOrder)
}

// SelectRunnerSuites filters suites by group and exact suite ID filters.
func SelectRunnerSuites(suites []RunnerSuite, groups, filters []string) ([]RunnerSuite, error) {
	groups = dedupeStrings(groups)
	filters = dedupeStrings(filters)
	for _, g := range groups {
		if _, ok := validRunnerGroups[g]; !ok {
			return nil, apperr.New(apperr.Usage, "conformance.runner", g, "unknown group")
		}
	}
	byID := map[string]RunnerSuite{}
	for _, s := range suites {
		byID[s.ID] = s
	}
	for _, id := range filters {
		if _, ok := byID[id]; !ok {
			return nil, apperr.New(apperr.Usage, "conformance.runner", id, "unknown suite id")
		}
	}
	var selected []RunnerSuite
	if len(groups) == 0 && len(filters) == 0 {
		selected = append(selected, suites...)
	} else if len(groups) > 0 && len(filters) == 0 {
		groupSet := map[string]struct{}{}
		for _, g := range groups {
			groupSet[g] = struct{}{}
		}
		for _, s := range suites {
			if _, ok := groupSet[s.Group]; ok {
				selected = append(selected, s)
			}
		}
	} else if len(groups) == 0 && len(filters) > 0 {
		for _, id := range filters {
			selected = append(selected, byID[id])
		}
	} else {
		groupSet := map[string]struct{}{}
		for _, g := range groups {
			groupSet[g] = struct{}{}
		}
		filterSet := map[string]struct{}{}
		for _, id := range filters {
			filterSet[id] = struct{}{}
		}
		for _, s := range suites {
			if _, ok := groupSet[s.Group]; !ok {
				continue
			}
			if _, ok := filterSet[s.ID]; ok {
				selected = append(selected, s)
			}
		}
	}
	sortRunnerSuites(selected)
	if len(selected) == 0 {
		return nil, apperr.New(apperr.Usage, "conformance.runner", "", "zero suites selected")
	}
	return selected, nil
}

func sortRunnerSuites(suites []RunnerSuite) {
	sort.SliceStable(suites, func(i, j int) bool {
		gi := groupIndex(suites[i].Group)
		gj := groupIndex(suites[j].Group)
		if gi != gj {
			return gi < gj
		}
		return suites[i].ID < suites[j].ID
	})
}

func dedupeStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func runnerSuiteSupportedOnPlatform(s RunnerSuite, goos string) bool {
	for _, p := range s.Platforms {
		if platformToGOOS(p) == goos {
			return true
		}
	}
	return false
}

func platformToGOOS(platform string) string {
	switch platform {
	case "linux":
		return "linux"
	case "darwin":
		return "darwin"
	case "windows":
		return "windows"
	default:
		return platform
	}
}

func goosToPlatform(goos string) string {
	switch goos {
	case "linux":
		return "linux"
	case "darwin":
		return "darwin"
	case "windows":
		return "windows"
	default:
		return goos
	}
}

// ValidateExpectedTestsRegex proves the run regex matches exactly expectedTests.
func ValidateExpectedTestsRegex(repoRoot string, suite RunnerSuite) error {
	matched, err := listTestsForSuite(repoRoot, suite, nil)
	if err != nil {
		return err
	}
	want := append([]string(nil), suite.ExpectedTests...)
	sort.Strings(want)
	sort.Strings(matched)
	if len(matched) == 0 {
		return fmt.Errorf("zero tests matched run regex")
	}
	if len(matched) != len(want) {
		return fmt.Errorf("regex matched %d tests, expected %d", len(matched), len(want))
	}
	for i := range want {
		if want[i] != matched[i] {
			return fmt.Errorf("regex match mismatch: got %v want %v", matched, want)
		}
	}
	return nil
}
