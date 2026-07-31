package app

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/conformance"
)

const runnerBenchCommandVersion = "runner-bench-v1"

// RunnerBenchProfile selects the benchmark case set.
type RunnerBenchProfile string

const (
	RunnerBenchProfileSmoke RunnerBenchProfile = "smoke"
	RunnerBenchProfileFull  RunnerBenchProfile = "full"
)

// RunnerBenchOptions configures m benchmark runner.
type RunnerBenchOptions struct {
	Profile RunnerBenchProfile
	CaseID  string
	Compare string
	Output  string
	Force   bool
	Samples int
}

// RunnerBenchBaselineV1 is the baseline schema for runner benchmarks.
type RunnerBenchBaselineV1 struct {
	SchemaVersion  int                    `json:"schemaVersion"`
	FixtureDigest  string                 `json:"fixtureDigest"`
	CommandVersion string                 `json:"commandVersion"`
	Environment    RunnerBenchEnvironment `json:"environment"`
	RecordedAt     string                 `json:"recordedAt"`
	Cases          []RunnerBenchCase      `json:"cases"`
}

// RunnerBenchEnvironment records host metadata for baselines.
type RunnerBenchEnvironment struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	MachineClass string `json:"machineClass"`
	GoVersion    string `json:"goVersion"`
	NodeVersion  string `json:"nodeVersion"`
}

// RunnerBenchCase is one benchmark measurement set.
type RunnerBenchCase struct {
	ID         string `json:"id"`
	CacheState string `json:"cacheState"`
	Samples    int    `json:"samples"`
	MedianNs   int64  `json:"medianNs"`
	P95Ns      int64  `json:"p95Ns"`
}

// RunnerBenchResult is the JSON output for m benchmark runner.
type RunnerBenchResult struct {
	SchemaVersion  int                    `json:"schemaVersion"`
	Profile        string                 `json:"profile,omitempty"`
	CaseID         string                 `json:"caseId,omitempty"`
	CommandVersion string                 `json:"commandVersion"`
	Environment    RunnerBenchEnvironment `json:"environment"`
	RecordedAt     string                 `json:"recordedAt"`
	Cases          []RunnerBenchCase      `json:"cases"`
	Compare        *RunnerBenchCompare    `json:"compare,omitempty"`
}

// RunnerBenchCompare records baseline comparison outcome.
type RunnerBenchCompare struct {
	Status   string `json:"status"`
	Baseline string `json:"baseline,omitempty"`
	Message  string `json:"message,omitempty"`
}

// BenchRunner executes runner smoke/full benchmarks without public registry access.
func BenchRunner(ctx context.Context, ac *Context, opts RunnerBenchOptions) (RunnerBenchResult, error) {
	if ac == nil {
		return RunnerBenchResult{}, apperr.New(apperr.Internal, "app.bench.runner", "", "missing app context")
	}
	if opts.Profile == "" {
		opts.Profile = RunnerBenchProfileSmoke
	}
	if opts.Profile != "" && opts.CaseID != "" {
		return RunnerBenchResult{}, apperr.New(apperr.Usage, "app.bench.runner", "", "--case and --profile are mutually exclusive")
	}
	cases := runnerBenchCases(opts)
	if len(cases) == 0 {
		return RunnerBenchResult{}, apperr.New(apperr.Usage, "app.bench.runner", "", "no benchmark cases selected")
	}
	samples := opts.Samples
	if samples <= 0 {
		samples = 5
	}
	repoRoot, err := conformance.RepoRootFromModule("")
	if err != nil {
		return RunnerBenchResult{}, apperr.Wrap(apperr.Internal, "app.bench.runner", "", err)
	}
	srcFixture := filepath.Join(repoRoot, "fixtures", "runner", "basic-scripts")
	fixtureRoot, err := os.MkdirTemp("", "mew-runner-bench-")
	if err != nil {
		return RunnerBenchResult{}, apperr.Wrap(apperr.IO, "app.bench.runner", "", err)
	}
	defer func() { _ = os.RemoveAll(fixtureRoot) }()
	if err := copyDir(srcFixture, fixtureRoot); err != nil {
		return RunnerBenchResult{}, apperr.Wrap(apperr.IO, "app.bench.runner", fixtureRoot, err)
	}
	measured := make([]RunnerBenchCase, 0, len(cases))
	for _, c := range cases {
		ns, err := measureRunnerCase(ctx, fixtureRoot, c.ID, samples)
		if err != nil {
			return RunnerBenchResult{}, err
		}
		sort.Slice(ns, func(i, j int) bool { return ns[i] < ns[j] })
		measured = append(measured, RunnerBenchCase{
			ID:         c.ID,
			CacheState: c.CacheState,
			Samples:    len(ns),
			MedianNs:   ns[len(ns)/2],
			P95Ns:      ns[int(float64(len(ns)-1)*0.95)],
		})
	}
	result := RunnerBenchResult{
		SchemaVersion:  1,
		Profile:        string(opts.Profile),
		CaseID:         opts.CaseID,
		CommandVersion: runnerBenchCommandVersion,
		Environment:    runnerBenchEnvironment(),
		RecordedAt:     time.Now().UTC().Format(time.RFC3339),
		Cases:          measured,
	}
	if opts.Compare != "" {
		cmp, err := compareRunnerBaseline(opts.Compare, result)
		if err != nil {
			return RunnerBenchResult{}, err
		}
		result.Compare = &cmp
	}
	if opts.Output != "" {
		if !opts.Force {
			if _, err := os.Stat(opts.Output); err == nil {
				return RunnerBenchResult{}, apperr.New(apperr.Usage, "app.bench.runner", opts.Output, "output file exists (use --force)")
			}
		}
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return RunnerBenchResult{}, apperr.Wrap(apperr.IO, "app.bench.runner", opts.Output, err)
		}
		tmp := opts.Output + ".tmp"
		if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
			return RunnerBenchResult{}, apperr.Wrap(apperr.IO, "app.bench.runner", opts.Output, err)
		}
		if err := os.Rename(tmp, opts.Output); err != nil {
			return RunnerBenchResult{}, apperr.Wrap(apperr.IO, "app.bench.runner", opts.Output, err)
		}
	}
	return result, nil
}

func runnerBenchCases(opts RunnerBenchOptions) []RunnerBenchCase {
	if opts.CaseID != "" {
		return []RunnerBenchCase{{ID: opts.CaseID, CacheState: "warm"}}
	}
	switch opts.Profile {
	case RunnerBenchProfileFull:
		return []RunnerBenchCase{
			{ID: "project-script", CacheState: "project"},
			{ID: "dlx-warm", CacheState: "warm"},
		}
	default:
		return []RunnerBenchCase{{ID: "project-script", CacheState: "project"}}
	}
}

func measureRunnerCase(ctx context.Context, fixtureRoot, caseID string, samples int) ([]int64, error) {
	if _, err := exec.LookPath("node"); err != nil {
		return nil, apperr.Wrap(apperr.NotFound, "app.bench.runner", "node", err)
	}
	repoRoot, err := conformance.RepoRootFromModule("")
	if err != nil {
		return nil, err
	}
	out := make([]int64, 0, samples)
	for i := 0; i < samples; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		start := time.Now()
		cmd := exec.CommandContext(ctx, "go", "run", filepath.Join(repoRoot, "cmd", "m"),
			"--cwd", fixtureRoot, "--output", "silent", "run", "dev")
		cmd.Dir = repoRoot
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		if err := cmd.Run(); err != nil {
			return nil, apperr.Wrap(apperr.Internal, "app.bench.runner", caseID, err)
		}
		out = append(out, time.Since(start).Nanoseconds())
	}
	return out, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = in.Close() }()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer func() { _ = out.Close() }()
		_, err = io.Copy(out, in)
		return err
	})
}

func runnerBenchEnvironment() RunnerBenchEnvironment {
	nodeVer := ""
	if path, err := exec.LookPath("node"); err == nil {
		cmd := exec.Command("node", "-v")
		cmd.Path = path
		if b, err := cmd.Output(); err == nil {
			nodeVer = strings.TrimSpace(string(b))
		}
	}
	return RunnerBenchEnvironment{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		MachineClass: runtime.GOARCH,
		GoVersion:    runtime.Version(),
		NodeVersion:  nodeVer,
	}
}

func compareRunnerBaseline(path string, result RunnerBenchResult) (RunnerBenchCompare, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RunnerBenchCompare{}, apperr.Wrap(apperr.Manifest, "app.bench.runner", path, err)
	}
	var baseline RunnerBenchBaselineV1
	if err := json.Unmarshal(data, &baseline); err != nil {
		return RunnerBenchCompare{}, apperr.Wrap(apperr.Manifest, "app.bench.runner", path, err)
	}
	if baseline.SchemaVersion != 1 || baseline.CommandVersion != runnerBenchCommandVersion {
		return RunnerBenchCompare{Status: "not-comparable", Baseline: path, Message: "incompatible baseline schema or command version"}, nil
	}
	return RunnerBenchCompare{Status: "comparable", Baseline: path}, nil
}

// EncodeRunnerBenchResultJSON returns indented JSON for a runner benchmark result.
func EncodeRunnerBenchResultJSON(r RunnerBenchResult) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
