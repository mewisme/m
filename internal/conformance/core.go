package conformance

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// RunOptions configures a core certification run.
type RunOptions struct {
	RepoRoot string
	Filter   string
	DryRun   bool
}

// RunCore executes the core certification matrix and returns a report.
func RunCore(ctx context.Context, opts RunOptions) (Report, error) {
	repoRoot := opts.RepoRoot
	if repoRoot == "" {
		var err error
		repoRoot, err = RepoRootFromModule("")
		if err != nil {
			return Report{}, err
		}
	}

	manifest, err := LoadManifest(CoreManifestPath(repoRoot))
	if err != nil {
		return Report{}, err
	}

	suites := FilterSuites(manifest.Suites, opts.Filter)
	suites = excludeProbeSuitesUnlessFiltered(suites, opts.Filter)
	if opts.Filter != "" && len(suites) == 0 {
		return Report{}, fmt.Errorf("no suites match filter %q", opts.Filter)
	}

	report := Report{
		SchemaVersion: ReportSchemaVersion,
		Matrix:        manifest.Matrix,
		CommitSHA:     ResolveCommitSHA(repoRoot),
		GoVersion:     runtime.Version(),
		StartedAt:     time.Now().UTC(),
		Filter:        opts.Filter,
		DryRun:        opts.DryRun,
		Suites:        make([]SuiteResult, 0, len(suites)),
	}
	if !opts.DryRun {
		report.Tools = CollectTools()
	}

	for _, suite := range suites {
		if !suiteSupportedOnPlatform(suite) {
			report.Suites = append(report.Suites, SuiteResult{
				ID:         suite.ID,
				Title:      suite.Title,
				Package:    suite.Package,
				Run:        suite.Run,
				Required:   suite.Required,
				Status:     StatusSkipped,
				SkipReason: "unsupported platform",
			})
			continue
		}
		if opts.DryRun {
			report.Suites = append(report.Suites, SuiteResult{
				ID:       suite.ID,
				Title:    suite.Title,
				Package:  suite.Package,
				Run:      suite.Run,
				Required: suite.Required,
				Status:   StatusPlanned,
			})
			continue
		}

		started := time.Now()
		exitCode, summary, output, runErr := RunTest(ctx, repoRoot, suite)
		report.Suites = append(report.Suites, suiteResultFromRun(suite, started, exitCode, summary, output, runErr))
	}

	report.FinishedAt = time.Now().UTC()
	report.Passed = reportPassed(report.Suites, opts.DryRun, opts.Filter)
	if !report.Passed && !opts.DryRun {
		return report, fmt.Errorf("core certification failed")
	}
	return report, nil
}

func reportPassed(suites []SuiteResult, dryRun bool, filter string) bool {
	explicitFilter := strings.TrimSpace(filter) != ""
	for _, s := range suites {
		if dryRun {
			if s.Status != StatusPlanned && s.Status != StatusSkipped {
				return false
			}
			continue
		}
		if explicitFilter {
			if s.Status != StatusPassed && s.Status != StatusSkipped {
				return false
			}
			continue
		}
		if !s.Required {
			continue
		}
		if s.Status != StatusPassed && s.Status != StatusSkipped {
			return false
		}
	}
	return true
}

func excludeProbeSuitesUnlessFiltered(suites []Suite, filter string) []Suite {
	if strings.TrimSpace(filter) != "" {
		return suites
	}
	var out []Suite
	for _, s := range suites {
		if !s.Probe {
			out = append(out, s)
		}
	}
	return out
}

// ListCore returns suite definitions from the core manifest, optionally filtered.
func ListCore(repoRoot, filter string) ([]Suite, error) {
	if repoRoot == "" {
		var err error
		repoRoot, err = RepoRootFromModule("")
		if err != nil {
			return nil, err
		}
	}
	manifest, err := LoadManifest(CoreManifestPath(repoRoot))
	if err != nil {
		return nil, err
	}
	suites := FilterSuites(manifest.Suites, filter)
	suites = excludeProbeSuitesUnlessFiltered(suites, filter)
	if filter != "" && len(suites) == 0 {
		return nil, fmt.Errorf("no suites match filter %q", filter)
	}
	return suites, nil
}
