package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

const installBaselineSchemaVersion = 2

// InstallBaselineCase is one published install benchmark case.
type InstallBaselineCase struct {
	Name          string  `json:"name"`
	TotalMsMedian int64   `json:"totalMsMedian"`
	TotalMsP95    int64   `json:"totalMsP95"`
	Samples       []int64 `json:"samples,omitempty"`
	GoVersion     string  `json:"goVersion,omitempty"`
	OS            string  `json:"os,omitempty"`
	Arch          string  `json:"arch,omitempty"`
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
		Commit:        result.Commit,
		FixtureDigest: result.FixtureDigest,
	}
	updated := false
	for i := range baseline.Cases {
		if baseline.Cases[i].Name == entry.Name {
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
	sort.Slice(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
}
