package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
)

// BenchMode selects cold (empty cache) or warm (reuse bench home cache).
type BenchMode string

const (
	BenchCold BenchMode = "cold"
	BenchWarm BenchMode = "warm"
)

// BenchInstallOptions configures m bench install.
type BenchInstallOptions struct {
	Fixture  string
	Mode     BenchMode
	Warmup   int
	Samples  int
	Baseline bool
}

// BenchResult is the JSON payload for m bench install --json.
type BenchResult struct {
	Case          string           `json:"case"`
	Mode          string           `json:"mode"`
	Samples       []int64          `json:"samples"`
	MedianMs      int64            `json:"medianMs"`
	P95Ms         int64            `json:"p95Ms"`
	TotalMs       int64            `json:"totalMs"`
	Phases        map[string]int64 `json:"phases,omitempty"`
	GoVersion     string           `json:"goVersion"`
	OS            string           `json:"os"`
	Arch          string           `json:"arch"`
	Commit        string           `json:"commit"`
	FixtureDigest string           `json:"fixtureDigest"`
	RunnerClass   string           `json:"runnerClass,omitempty"`
}

type phaseReporter struct {
	inner      diagnostics.Reporter
	start      time.Time
	phaseStart time.Time
	lastPhase  string
	phases     map[string]int64
}

func newPhaseReporter(inner diagnostics.Reporter) *phaseReporter {
	now := time.Now()
	return &phaseReporter{
		inner:      inner,
		start:      now,
		phaseStart: now,
		phases:     map[string]int64{},
	}
}

func (p *phaseReporter) Progress(ev diagnostics.Event) {
	if ev.Type == "" || ev.Type == "progress" {
		if phase := strings.TrimSpace(ev.Phase); phase != "" && phase != p.lastPhase {
			now := time.Now()
			if p.lastPhase != "" {
				p.phases[p.lastPhase] += now.Sub(p.phaseStart).Milliseconds()
			}
			p.lastPhase = phase
			p.phaseStart = now
		}
	}
	if p.inner != nil {
		p.inner.Progress(ev)
	}
}

func (p *phaseReporter) Error(err error) {
	if p.inner != nil {
		p.inner.Error(err)
	}
}

func (p *phaseReporter) Debug(msg string, attrs ...diagnostics.Attr) {
	if p.inner != nil {
		p.inner.Debug(msg, attrs...)
	}
}

func (p *phaseReporter) WorkspaceTask(ev diagnostics.WorkspaceTaskEvent) {
	if p.inner != nil {
		p.inner.WorkspaceTask(ev)
	}
}

func (p *phaseReporter) ChildOutput(ev diagnostics.ChildOutputEvent, mode diagnostics.WorkspaceOutputMode) {
	if p.inner != nil {
		p.inner.ChildOutput(ev, mode)
	}
}

func (p *phaseReporter) WorkspaceSummary(ev diagnostics.WorkspaceSummaryEvent) {
	if p.inner != nil {
		p.inner.WorkspaceSummary(ev)
	}
}

func (p *phaseReporter) EnvironmentPrepared(ev diagnostics.EnvironmentPreparedEvent) error {
	if p.inner != nil {
		return p.inner.EnvironmentPrepared(ev)
	}
	return nil
}

func (p *phaseReporter) OperationStarted(ev diagnostics.OperationStartedEvent) {
	if p.inner != nil {
		p.inner.OperationStarted(ev)
	}
}

func (p *phaseReporter) OperationProgress(ev diagnostics.OperationProgressEvent) {
	if p.inner != nil {
		p.inner.OperationProgress(ev)
	}
}

func (p *phaseReporter) OperationCompleted(ev diagnostics.OperationCompletedEvent) {
	if p.inner != nil {
		p.inner.OperationCompleted(ev)
	}
}

func (p *phaseReporter) Notice(ev diagnostics.NoticeEvent) {
	if p.inner != nil {
		p.inner.Notice(ev)
	}
}

func (p *phaseReporter) finish() map[string]int64 {
	now := time.Now()
	if p.lastPhase != "" {
		p.phases[p.lastPhase] += now.Sub(p.phaseStart).Milliseconds()
	}
	out := make(map[string]int64, len(p.phases))
	for k, v := range p.phases {
		out[k] = v
	}
	return out
}

func (p *phaseReporter) totalMs() int64 {
	return time.Since(p.start).Milliseconds()
}

type benchSample struct {
	totalMs int64
	phases  map[string]int64
}

// BenchInstall runs an isolated install benchmark against a fixture project.
func BenchInstall(ctx context.Context, ac *Context, opts BenchInstallOptions) (BenchResult, error) {
	if ac == nil {
		return BenchResult{}, apperr.New(apperr.Internal, "app.bench", "", "missing app context")
	}
	mode := opts.Mode
	if mode == "" {
		mode = BenchCold
	}
	if mode != BenchCold && mode != BenchWarm {
		return BenchResult{}, apperr.New(apperr.Usage, "app.bench", string(mode), "mode must be cold or warm")
	}
	fixtureRel := strings.TrimSpace(opts.Fixture)
	if fixtureRel == "" {
		fixtureRel = "fixtures/bench/medium-graph"
	}
	moduleRoot, err := findModuleRoot(ac.CWD)
	if err != nil {
		return BenchResult{}, apperr.Wrap(apperr.Usage, "app.bench", fixtureRel, err)
	}
	fixtureSrc, err := resolveBenchFixture(moduleRoot, ac.CWD, fixtureRel)
	if err != nil {
		return BenchResult{}, err
	}
	fixtureDigest, err := fixtureTreeDigest(fixtureSrc)
	if err != nil {
		return BenchResult{}, apperr.Wrap(apperr.IO, "app.bench", fixtureSrc, err)
	}
	caseName := filepath.Base(fixtureSrc) + "-" + string(mode)
	goVersion, goos, goarch := benchRuntimeMetadata(ac.Commit)

	warmup := benchWarmupCount(opts.Warmup)
	samples := benchSampleCount(opts.Samples)
	var totals []int64
	var lastPhases map[string]int64
	for i := 0; i < warmup+samples; i++ {
		sample, err := benchInstallOnce(ctx, ac, mode, fixtureSrc, moduleRoot)
		if err != nil {
			return BenchResult{}, err
		}
		if i < warmup {
			continue
		}
		totals = append(totals, sample.totalMs)
		lastPhases = sample.phases
	}

	median := benchMedian(totals)
	p95 := benchP95(totals)
	result := BenchResult{
		Case:          caseName,
		Mode:          string(mode),
		Samples:       totals,
		MedianMs:      median,
		P95Ms:         p95,
		TotalMs:       median,
		Phases:        lastPhases,
		GoVersion:     goVersion,
		OS:            goos,
		Arch:          goarch,
		Commit:        ac.Commit,
		FixtureDigest: fixtureDigest,
		RunnerClass:   benchRunnerClass(),
	}
	if opts.Baseline {
		if err := UpdateInstallBaseline(moduleRoot, result); err != nil {
			return BenchResult{}, err
		}
	}
	return result, nil
}

func benchInstallOnce(ctx context.Context, ac *Context, mode BenchMode, fixtureSrc, moduleRoot string) (benchSample, error) {
	benchRoot := filepath.Join(moduleRoot, ".cache", "mew", "bench", filepath.Base(fixtureSrc))
	home := filepath.Join(benchRoot, "home")
	projectDir := filepath.Join(benchRoot, "project")
	if err := os.MkdirAll(benchRoot, 0o755); err != nil {
		return benchSample{}, apperr.Wrap(apperr.IO, "app.bench", benchRoot, err)
	}

	if mode == BenchCold {
		if err := os.RemoveAll(home); err != nil {
			return benchSample{}, apperr.Wrap(apperr.IO, "app.bench", home, err)
		}
	}
	if err := os.RemoveAll(projectDir); err != nil {
		return benchSample{}, apperr.Wrap(apperr.IO, "app.bench", projectDir, err)
	}
	if err := copyBenchTree(fixtureSrc, projectDir); err != nil {
		return benchSample{}, apperr.Wrap(apperr.IO, "app.bench", fixtureSrc, err)
	}

	cacheDir := filepath.Join(home, ".cache", "mew")
	storeDir := filepath.Join(home, ".local", "share", "github.com", "mewisme", "mew", "store")
	configDir := filepath.Join(home, ".config", "mew")
	for _, d := range []string{cacheDir, storeDir, configDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return benchSample{}, apperr.Wrap(apperr.IO, "app.bench", d, err)
		}
	}

	registryRoot := filepath.Join(moduleRoot, "fixtures", "registry", "v1")
	reg, err := startFixtureRegistry(registryRoot)
	if err != nil {
		return benchSample{}, apperr.Wrap(apperr.Network, "app.bench", registryRoot, err)
	}
	defer reg.Close()

	cfgPath := filepath.Join(projectDir, "m.jsonc")
	cfgBody := fmt.Sprintf(`{"registry":"%s"}`+"\n", reg.URL())
	if err := os.WriteFile(cfgPath, []byte(cfgBody), 0o644); err != nil {
		return benchSample{}, apperr.Wrap(apperr.IO, "app.bench", cfgPath, err)
	}

	env := benchOverlayEnv(os.Environ(), home, cacheDir, storeDir, configDir)
	phaseRep := newPhaseReporter(ac.Reporter)
	benchAC, err := New(ctx, Options{
		CWD:        projectDir,
		ConfigPath: cfgPath,
		Env:        env,
		Reporter:   phaseRep,
		Version:    ac.Version,
		Commit:     ac.Commit,
		BuildDate:  ac.BuildDate,
	})
	if err != nil {
		return benchSample{}, err
	}

	_, err = Install(ctx, benchAC, InstallOptions{IgnoreScripts: true})
	if err != nil {
		return benchSample{}, err
	}

	return benchSample{
		totalMs: phaseRep.totalMs(),
		phases:  phaseRep.finish(),
	}, nil
}

func benchOverlayEnv(base []string, home, cacheDir, storeDir, configDir string) []string {
	out := make([]string, 0, len(base)+12)
	for _, kv := range base {
		key := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			key = kv[:i]
		}
		switch key {
		case "HOME", "USERPROFILE", "XDG_CACHE_HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME",
			"MEW_HOME", "MEW_CACHE_DIR", "MEW_STORE_DIR", "MEW_CONFIG_DIR", "NO_PROXY", "no_proxy":
			continue
		}
		out = append(out, kv)
	}
	out = append(out,
		"HOME="+home,
		"USERPROFILE="+home,
		"XDG_CACHE_HOME="+filepath.Join(home, ".cache"),
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME="+filepath.Join(home, ".local", "state"),
		"MEW_HOME="+home,
		"MEW_CACHE_DIR="+cacheDir,
		"MEW_STORE_DIR="+storeDir,
		"MEW_CONFIG_DIR="+configDir,
		"NO_PROXY=*",
		"no_proxy=*",
	)
	return out
}

func findModuleRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", start)
		}
		dir = parent
	}
}

func resolveBenchFixture(moduleRoot, cwd, fixtureRel string) (string, error) {
	if filepath.IsAbs(fixtureRel) {
		if _, err := os.Stat(fixtureRel); err != nil {
			return "", apperr.Wrap(apperr.NotFound, "app.bench", fixtureRel, err)
		}
		return fixtureRel, nil
	}
	candidates := []string{
		filepath.Join(cwd, fixtureRel),
		filepath.Join(moduleRoot, fixtureRel),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", apperr.New(apperr.NotFound, "app.bench", fixtureRel, "fixture not found")
}

func copyBenchTree(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = in.Close() }()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		defer func() { _ = out.Close() }()
		_, err = io.Copy(out, in)
		return err
	})
}

// EncodeBenchResultJSON marshals a bench result for CLI output.
func EncodeBenchResultJSON(r BenchResult) ([]byte, error) {
	return json.Marshal(r)
}
