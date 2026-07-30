package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
)

// RunnerVerifyOptions configures cross-platform report aggregation.
type RunnerVerifyOptions struct {
	ReportPaths []string
	OutputPath  string
}

// RunnerCertificationSummary is the aggregated certification result.
type RunnerCertificationSummary struct {
	SchemaVersion        int                          `json:"schemaVersion"`
	Matrix               string                       `json:"matrix"`
	ManifestDigest       string                       `json:"manifestDigest"`
	WaiverManifestDigest string                       `json:"waiverManifestDigest"`
	Commit               string                       `json:"commit"`
	PlatformReports      []RunnerPlatformReportDigest `json:"platformReports"`
	Coverage             []RunnerCoverageEntry        `json:"coverage"`
	Waivers              []string                     `json:"waivers,omitempty"`
	Overall              string                       `json:"overall"`
}

// RunnerPlatformReportDigest records one platform report fingerprint.
type RunnerPlatformReportDigest struct {
	Platform             string `json:"platform"`
	Certification        string `json:"certification"`
	ReportDigest         string `json:"reportDigest"`
	Commit               string `json:"commit"`
	ManifestDigest       string `json:"manifestDigest"`
	WaiverManifestDigest string `json:"waiverManifestDigest"`
}

// RunnerCoverageEntry records per-suite platform coverage.
type RunnerCoverageEntry struct {
	SuiteID   string            `json:"suiteId"`
	Platforms map[string]string `json:"platforms"`
}

// VerifyRunnerReports aggregates per-platform runner reports.
func VerifyRunnerReports(opts RunnerVerifyOptions) (RunnerCertificationSummary, error) {
	if len(opts.ReportPaths) == 0 {
		return RunnerCertificationSummary{}, apperr.New(apperr.Usage, "conformance.verify.runner", "", "at least one --report required")
	}
	reports := make([]RunnerReport, 0, len(opts.ReportPaths))
	for _, path := range opts.ReportPaths {
		r, err := loadRunnerReport(path)
		if err != nil {
			return RunnerCertificationSummary{}, err
		}
		reports = append(reports, r)
	}
	manifestDigest := reports[0].ManifestDigest
	waiverDigest := reports[0].WaiverManifestDigest
	commit := reports[0].Revision.Commit
	for _, r := range reports[1:] {
		if r.SchemaVersion != RunnerReportSchemaVersion {
			return RunnerCertificationSummary{}, apperr.New(apperr.Manifest, "conformance.verify.runner", "", "unsupported report schema")
		}
		if r.Matrix != RunnerMatrix {
			return RunnerCertificationSummary{}, apperr.New(apperr.Manifest, "conformance.verify.runner", "", "matrix mismatch")
		}
		if r.ManifestDigest != manifestDigest {
			return RunnerCertificationSummary{}, fmt.Errorf("manifest digest mismatch")
		}
		if r.WaiverManifestDigest != waiverDigest {
			return RunnerCertificationSummary{}, fmt.Errorf("waiver digest mismatch")
		}
		if r.Revision.Commit != commit {
			return RunnerCertificationSummary{}, fmt.Errorf("commit mismatch across platform reports")
		}
		if r.Revision.Dirty {
			return RunnerCertificationSummary{}, fmt.Errorf("dirty report from %s", r.Environment.OS)
		}
	}
	summary := RunnerCertificationSummary{
		SchemaVersion:        RunnerReportSchemaVersion,
		Matrix:               RunnerMatrix,
		ManifestDigest:       manifestDigest,
		WaiverManifestDigest: waiverDigest,
		Commit:               commit,
		PlatformReports:      make([]RunnerPlatformReportDigest, 0, len(reports)),
		Coverage:             buildCoverageMatrix(reports),
		Overall:              RunnerResultPass,
	}
	requiredPlatforms := map[string]struct{}{"linux": {}, "windows": {}}
	seenPlatforms := map[string]struct{}{}
	for i, r := range reports {
		platform := r.Environment.OS
		seenPlatforms[platform] = struct{}{}
		digest, err := reportFileDigest(opts.ReportPaths[i])
		if err != nil {
			return RunnerCertificationSummary{}, err
		}
		cert := "ci-certified"
		if platform == "darwin" && os.Getenv("MEW_RUNNER_LOCAL_EVIDENCE") == "1" {
			cert = "locally-certified"
		}
		summary.PlatformReports = append(summary.PlatformReports, RunnerPlatformReportDigest{
			Platform:             platform,
			Certification:        cert,
			ReportDigest:         digest,
			Commit:               r.Revision.Commit,
			ManifestDigest:       r.ManifestDigest,
			WaiverManifestDigest: r.WaiverManifestDigest,
		})
		if r.Overall != RunnerResultPass {
			summary.Overall = RunnerResultFail
		}
		for _, s := range r.Suites {
			if s.Required && s.Result != RunnerResultPass && s.Result != RunnerResultPassWithWaiver && s.Result != RunnerResultNotApplicable && s.Result != RunnerResultProbeSkip {
				summary.Overall = RunnerResultFail
			}
			if s.WaiverPolicy == "forbidden" && len(s.AppliedWaiverIDs) > 0 {
				summary.Overall = RunnerResultFail
			}
			if len(s.SkippedTests) > 0 && s.Required && !s.Probe {
				summary.Overall = RunnerResultFail
			}
		}
	}
	for p := range requiredPlatforms {
		if _, ok := seenPlatforms[p]; !ok {
			summary.Overall = RunnerResultFail
		}
	}
	if summary.Overall != RunnerResultPass {
		return summary, fmt.Errorf("runner certification summary failed")
	}
	if opts.OutputPath != "" {
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return summary, apperr.Wrap(apperr.IO, "conformance.verify.runner", opts.OutputPath, err)
		}
		if err := os.WriteFile(opts.OutputPath, append(data, '\n'), 0o644); err != nil {
			return summary, apperr.Wrap(apperr.IO, "conformance.verify.runner", opts.OutputPath, err)
		}
	}
	return summary, nil
}

func loadRunnerReport(path string) (RunnerReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RunnerReport{}, apperr.Wrap(apperr.NotFound, "conformance.verify.runner", path, err)
	}
	var r RunnerReport
	if err := json.Unmarshal(data, &r); err != nil {
		return RunnerReport{}, apperr.Wrap(apperr.Manifest, "conformance.verify.runner", path, err)
	}
	if r.SchemaVersion != RunnerReportSchemaVersion {
		return RunnerReport{}, apperr.New(apperr.Manifest, "conformance.verify.runner", path, "unsupported schema")
	}
	return r, nil
}

func buildCoverageMatrix(reports []RunnerReport) []RunnerCoverageEntry {
	bySuite := map[string]map[string]string{}
	for _, r := range reports {
		for _, s := range r.Suites {
			if _, ok := bySuite[s.ID]; !ok {
				bySuite[s.ID] = map[string]string{}
			}
			bySuite[s.ID][s.Platform] = s.Result
		}
	}
	ids := make([]string, 0, len(bySuite))
	for id := range bySuite {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]RunnerCoverageEntry, 0, len(ids))
	for _, id := range ids {
		out = append(out, RunnerCoverageEntry{SuiteID: id, Platforms: bySuite[id]})
	}
	return out
}

func reportFileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return canonicalDigestBytes(data)
}

func canonicalDigestBytes(data []byte) (string, error) {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
