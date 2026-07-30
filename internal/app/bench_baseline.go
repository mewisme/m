package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

const installBaselineSchemaVersion = 3

// InstallBaselineCase is one published install benchmark case.
type InstallBaselineCase struct {
	Name          string  `json:"name"`
	TotalMsMedian int64   `json:"totalMsMedian"`
	TotalMsP95    int64   `json:"totalMsP95"`
	Samples       []int64 `json:"samples,omitempty"`
	GoVersion     string  `json:"goVersion,omitempty"`
	OS            string  `json:"os,omitempty"`
	Arch          string  `json:"arch,omitempty"`
	RunnerClass   string  `json:"runnerClass,omitempty"`
	BenchmarkMode string  `json:"benchmarkMode,omitempty"`
	Commit        string  `json:"commit,omitempty"`
	FixtureDigest string  `json:"fixtureDigest,omitempty"`
}

// InstallBaseline is the on-disk install regression baseline file.
type InstallBaseline struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Cases         []InstallBaselineCase `json:"cases"`
}

func installBaselinePath(moduleRoot string) string {
	return filepath.Join(moduleRoot, "benchmarks", "install-baseline.json")
}

func baselineCaseKey(c InstallBaselineCase) string {
	parts := []string{
		c.Name,
		strings.ToLower(c.OS),
		strings.ToLower(c.Arch),
		c.GoVersion,
		strings.ToLower(c.RunnerClass),
		strings.ToLower(c.BenchmarkMode),
	}
	return strings.Join(parts, "|")
}

// MatchInstallBaselineCase selects a baseline row for the current host metadata.
func MatchInstallBaselineCase(baseline InstallBaseline, result BenchResult) (InstallBaselineCase, bool) {
	want := baselineCaseKey(InstallBaselineCase{
		Name:          result.Case,
		OS:            result.OS,
		Arch:          result.Arch,
		GoVersion:     result.GoVersion,
		RunnerClass:   result.RunnerClass,
		BenchmarkMode: result.Mode,
	})
	for _, c := range baseline.Cases {
		if baselineCaseKey(c) == want {
			return c, true
		}
	}
	// ponytail: legacy v2 rows keyed by name+os+arch+mode only when runnerClass empty.
	if result.RunnerClass == "" {
		for _, c := range baseline.Cases {
			if c.Name == result.Case &&
				strings.EqualFold(c.OS, result.OS) &&
				strings.EqualFold(c.Arch, result.Arch) &&
				strings.EqualFold(c.BenchmarkMode, result.Mode) &&
				c.RunnerClass == "" {
				return c, true
			}
		}
	}
	return InstallBaselineCase{}, false
}

// UpdateInstallBaseline merges one bench result into benchmarks/install-baseline.json.
func UpdateInstallBaseline(moduleRoot string, result BenchResult) error {
	path := installBaselinePath(moduleRoot)
	var baseline InstallBaseline
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &baseline); err != nil {
			return apperr.Wrap(apperr.IO, "app.bench.baseline", path, err)
		}
	} else if !os.IsNotExist(err) {
		return apperr.Wrap(apperr.IO, "app.bench.baseline", path, err)
	}
	baseline.SchemaVersion = installBaselineSchemaVersion
	entry := InstallBaselineCase{
		Name:          result.Case,
		TotalMsMedian: result.MedianMs,
		TotalMsP95:    result.P95Ms,
		Samples:       append([]int64(nil), result.Samples...),
		GoVersion:     result.GoVersion,
		OS:            result.OS,
		Arch:          result.Arch,
		RunnerClass:   result.RunnerClass,
		BenchmarkMode: result.Mode,
		Commit:        result.Commit,
		FixtureDigest: result.FixtureDigest,
	}
	key := baselineCaseKey(entry)
	updated := false
	for i := range baseline.Cases {
		if baselineCaseKey(baseline.Cases[i]) == key {
			baseline.Cases[i] = entry
			updated = true
			break
		}
	}
	if !updated {
		baseline.Cases = append(baseline.Cases, entry)
	}
	sortBaselineCases(baseline.Cases)
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "app.bench.baseline", path, err)
	}
	data = append(data, '\n')
	if err := fsx.PublishFileDurable(path, data, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "app.bench.baseline", path, err)
	}
	return nil
}

func sortBaselineCases(cases []InstallBaselineCase) {
	sort.Slice(cases, func(i, j int) bool { return baselineCaseKey(cases[i]) < baselineCaseKey(cases[j]) })
}
