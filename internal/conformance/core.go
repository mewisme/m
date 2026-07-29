package conformance

import (
	"context"
	"fmt"
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
	if opts.Filter != "" && len(suites) == 0 {
		return Report{}, fmt.Errorf("no suites match filter %q", opts.Filter)
	}

	report := Report{
		SchemaVersion: SchemaVersion,
		Matrix:        manifest.Matrix,
		StartedAt:     time.Now().UTC(),
		Filter:        opts.Filter,
		DryRun:        opts.DryRun,
		Suites:        make([]SuiteResult, 0, len(suites)),
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
				Skipped:    true,
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
		exitCode, output, runErr := RunTest(ctx, repoRoot, suite)
		report.Suites = append(report.Suites, suiteResultFromRun(suite, started, exitCode, output, runErr))
	}

	report.FinishedAt = time.Now().UTC()
	report.Passed = reportPassed(report.Suites, opts.DryRun)
	if !report.Passed && !opts.DryRun {
		return report, fmt.Errorf("core certification failed")
	}
	return report, nil
}

func reportPassed(suites []SuiteResult, dryRun bool) bool {
	for _, s := range suites {
		if !s.Required {
			continue
		}
		if dryRun {
			if s.Status != StatusPlanned && s.Status != StatusSkipped {
				return false
			}
			continue
		}
		if s.Status != StatusPassed && s.Status != StatusSkipped {
			return false
		}
	}
	return true
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
	if filter != "" && len(suites) == 0 {
		return nil, fmt.Errorf("no suites match filter %q", filter)
	}
	return suites, nil
}
