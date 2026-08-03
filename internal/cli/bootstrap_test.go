package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/presentation"
	"github.com/mewisme/mew/internal/testkit"
)

// bootstrapProbe records what one invocation loaded, created, and closed.
type bootstrapProbe struct {
	configLoads atomic.Int32
	controllers atomic.Int32
	closes      atomic.Int32
	loadOpts    []app.Options
	resolved    presentation.ResolvedOptions
}

// installProbe swaps in counting seams for the config loader and controller
// factory, restoring both when the test ends.
func installProbe(t *testing.T) *bootstrapProbe {
	t.Helper()
	p := &bootstrapProbe{}

	prevLoad, prevCtrl := loadConfigFn, newControllerFn
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		newControllerFn = prevCtrl
	})

	loadConfigFn = func(ctx context.Context, opts app.Options) (app.ConfigSnapshot, error) {
		p.configLoads.Add(1)
		p.loadOpts = append(p.loadOpts, opts)
		return prevLoad(ctx, opts)
	}
	newControllerFn = func(resolved presentation.ResolvedOptions, caps presentation.Capabilities, streams presentation.StreamWriters) (presentation.Controller, error) {
		p.controllers.Add(1)
		p.resolved = resolved
		ctrl, err := prevCtrl(resolved, caps, streams)
		if err != nil {
			return nil, err
		}
		return &countingController{Controller: ctrl, closes: &p.closes}, nil
	}
	return p
}

// countingController counts Close calls so lifecycle tests can prove the
// controller is closed exactly once.
type countingController struct {
	presentation.Controller
	closes *atomic.Int32
}

func (c *countingController) Close(ctx context.Context, outcome presentation.Outcome) error {
	c.closes.Add(1)
	return c.Controller.Close(ctx, outcome)
}

// probedRoot is an m root wired to a buffer plus recording commands.
type probedRoot struct {
	root *cobra.Command
	out  *bytes.Buffer
	ac   *app.Context
}

// newProbedRoot builds an m root with commands that record the app context they
// receive, fail, or panic on demand.
func newProbedRoot(t *testing.T) *probedRoot {
	t.Helper()
	pr := &probedRoot{root: NewMRoot(testBuildInfo()), out: new(bytes.Buffer)}
	pr.root.SetOut(pr.out)
	pr.root.SetErr(pr.out)
	pr.root.AddCommand(&cobra.Command{
		Use: "probe",
		RunE: func(cmd *cobra.Command, args []string) error {
			pr.ac = app.FromContext(cmd.Context())
			return nil
		},
	})
	pr.root.AddCommand(&cobra.Command{
		Use:  "probefail",
		RunE: func(cmd *cobra.Command, args []string) error { return apperr.New(apperr.Internal, "probe", "", "boom") },
	})
	pr.root.AddCommand(&cobra.Command{
		Use:  "probepanic",
		RunE: func(cmd *cobra.Command, args []string) error { panic("probe panic") },
	})
	return pr
}

// writeUserConfig writes the user-scope config file into the isolated home.
func writeUserConfig(t *testing.T, env testkit.CleanEnvInfo, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(env.ConfigDir, "config.jsonc"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ── §1 presentation flags reach the controller ──────────────────────────────

func TestBootstrapPresentationFlagsReachController(t *testing.T) {
	cases := []struct {
		name  string
		argv  []string
		check func(t *testing.T, r presentation.ResolvedOptions)
	}{
		{"output-json", []string{"--output=json", "probe"}, func(t *testing.T, r presentation.ResolvedOptions) {
			if r.Output != presentation.OutputJSON {
				t.Fatalf("Output=%q want json", r.Output)
			}
		}},
		{"output-plain", []string{"--output=plain", "probe"}, func(t *testing.T, r presentation.ResolvedOptions) {
			if r.Output != presentation.OutputPlain {
				t.Fatalf("Output=%q want plain", r.Output)
			}
		}},
		{"output-silent", []string{"--output=silent", "probe"}, func(t *testing.T, r presentation.ResolvedOptions) {
			if r.Output != presentation.OutputSilent {
				t.Fatalf("Output=%q want silent", r.Output)
			}
		}},
		{"no-color", []string{"--no-color", "probe"}, func(t *testing.T, r presentation.ResolvedOptions) {
			if r.Color {
				t.Fatal("Color must be false with --no-color")
			}
		}},
		{"accessible", []string{"--accessible", "probe"}, func(t *testing.T, r presentation.ResolvedOptions) {
			if !r.Accessible {
				t.Fatal("Accessible must be true")
			}
		}},
		{"no-progress", []string{"--no-progress", "probe"}, func(t *testing.T, r presentation.ResolvedOptions) {
			if r.Progress {
				t.Fatal("Progress must be false")
			}
		}},
		{"ascii", []string{"--ascii", "probe"}, func(t *testing.T, r presentation.ResolvedOptions) {
			if r.Unicode {
				t.Fatal("Unicode must be false with --ascii")
			}
		}},
		{"no-summary", []string{"--no-summary", "probe"}, func(t *testing.T, r presentation.ResolvedOptions) {
			if r.Summary {
				t.Fatal("Summary must be false")
			}
		}},
		{"log-level-debug", []string{"--log-level=debug", "probe"}, func(t *testing.T, r presentation.ResolvedOptions) {
			if r.LogLevel != presentation.LogDebug {
				t.Fatalf("LogLevel=%q want debug", r.LogLevel)
			}
		}},
		{"debug", []string{"--debug", "probe"}, func(t *testing.T, r presentation.ResolvedOptions) {
			if !r.Debug {
				t.Fatal("Debug must be true")
			}
		}},
		{"flag-after-builtin", []string{"probe", "--output=json"}, func(t *testing.T, r presentation.ResolvedOptions) {
			if r.Output != presentation.OutputJSON {
				t.Fatalf("Output=%q want json for flag after command", r.Output)
			}
		}},
		{"combined", []string{"--output=plain", "--ascii", "--no-summary", "probe"}, func(t *testing.T, r presentation.ResolvedOptions) {
			if r.Output != presentation.OutputPlain || r.Unicode || r.Summary {
				t.Fatalf("resolved=%+v want plain+ascii+no-summary", r)
			}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testkit.CleanEnv(t)
			p := installProbe(t)
			pr := newProbedRoot(t)

			if code := runInvocation(context.Background(), pr.root, testBuildInfo(), tc.argv); code != 0 {
				t.Fatalf("exit=%d out=%s", code, pr.out.String())
			}
			if got := p.controllers.Load(); got != 1 {
				t.Fatalf("controllers=%d want 1", got)
			}
			tc.check(t, p.resolved)
		})
	}
}

func TestBootstrapStopsAtDoubleDash(t *testing.T) {
	testkit.CleanEnv(t)
	p := installProbe(t)
	pr := newProbedRoot(t)

	// --output after -- belongs to the child, not to Mew.
	argv := []string{"probe", "--", "--output=json"}
	if code := runInvocation(context.Background(), pr.root, testBuildInfo(), argv); code != 0 {
		t.Fatalf("exit=%d out=%s", code, pr.out.String())
	}
	if p.resolved.Output == presentation.OutputJSON {
		t.Fatal("flags after -- must not configure Mew presentation")
	}
}

func TestBootstrapDoesNotConsumeChildArgs(t *testing.T) {
	testkit.CleanEnv(t)
	p := installProbe(t)
	pr := newProbedRoot(t)

	// "unknown-script" is not a builtin, so everything after it is child argv.
	argv := []string{"unknown-script", "--output=json", "--no-color"}
	_ = runInvocation(context.Background(), pr.root, testBuildInfo(), argv)
	if p.resolved.Output == presentation.OutputJSON {
		t.Fatal("script arguments must not configure Mew presentation")
	}
}

func TestBootstrapLocalFlagShadowsGlobal(t *testing.T) {
	testkit.CleanEnv(t)
	p := installProbe(t)
	pr := newProbedRoot(t)

	// plan --output takes a file path; it must not be read as an output mode.
	target := filepath.Join(t.TempDir(), "plan.json")
	_ = runInvocation(context.Background(), pr.root, testBuildInfo(), []string{"plan", "--output", target})
	if p.resolved.Output == presentation.OutputJSON || p.resolved.Output == presentation.OutputPlain {
		t.Fatalf("local --output leaked into presentation: %q", p.resolved.Output)
	}
	if got := p.controllers.Load(); got != 1 {
		t.Fatalf("controllers=%d want 1", got)
	}
}

func TestBootstrapPresentationParseFailureCreatesNoController(t *testing.T) {
	testkit.CleanEnv(t)
	p := installProbe(t)
	pr := newProbedRoot(t)

	code := runInvocation(context.Background(), pr.root, testBuildInfo(), []string{"--output=bogus", "probe"})
	if code != apperr.ExitCode(apperr.New(apperr.Usage, "", "", "")) {
		t.Fatalf("exit=%d want usage exit", code)
	}
	if got := p.controllers.Load(); got != 0 {
		t.Fatalf("controllers=%d want 0 on presentation failure", got)
	}
	if got := p.closes.Load(); got != 0 {
		t.Fatalf("closes=%d want 0 when no controller exists", got)
	}
	if !strings.Contains(pr.out.String(), "ERR_M_USAGE") {
		t.Fatalf("expected usage error, got %q", pr.out.String())
	}
}

// ── §3 ui.theme wiring ──────────────────────────────────────────────────────

func TestBootstrapUIThemeReachesRenderer(t *testing.T) {
	cases := []struct {
		name    string
		theme   string
		argv    []string
		want    presentation.ThemeMode
		wantErr bool
	}{
		{name: "light", theme: "light", argv: []string{"probe"}, want: presentation.ThemeLight},
		{name: "dark", theme: "dark", argv: []string{"probe"}, want: presentation.ThemeDark},
		{name: "no-color-overrides-light", theme: "light", argv: []string{"--no-color", "probe"}, want: presentation.ThemeNone},
		{name: "no-color-overrides-dark", theme: "dark", argv: []string{"--no-color", "probe"}, want: presentation.ThemeNone},
		{name: "invalid", theme: "neon", argv: []string{"probe"}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := testkit.CleanEnv(t)
			clearHelpEnv(t)
			writeUserConfig(t, env, `{"ui":{"theme":"`+tc.theme+`"}}`)
			p := installProbe(t)
			pr := newProbedRoot(t)

			code := runInvocation(context.Background(), pr.root, testBuildInfo(), tc.argv)
			if tc.wantErr {
				if code == 0 {
					t.Fatalf("invalid ui.theme must block a normal command, exit=%d", code)
				}
				if !strings.Contains(pr.out.String(), "ERR_M_CONFIG") {
					t.Fatalf("want typed config error, got %q", pr.out.String())
				}
				return
			}
			if code != 0 {
				t.Fatalf("exit=%d out=%s", code, pr.out.String())
			}
			// The controller settings carry the mode the renderer uses.
			got := presentation.Effective(p.resolved, presentation.Capabilities{
				StdoutTTY: true, StderrTTY: true,
				ColorProfile: presentation.ColorProfileANSI, Width: 80,
			}, nil).ThemeMode
			if got != tc.want {
				t.Fatalf("ThemeMode=%q want %q (ui.theme=%q)", got, tc.want, tc.theme)
			}
		})
	}
}

func TestBootstrapAutoThemeUnchanged(t *testing.T) {
	env := testkit.CleanEnv(t)
	clearHelpEnv(t)
	writeUserConfig(t, env, `{"ui":{"theme":"auto"}}`)
	p := installProbe(t)
	pr := newProbedRoot(t)

	if code := runInvocation(context.Background(), pr.root, testBuildInfo(), []string{"probe"}); code != 0 {
		t.Fatalf("exit=%d out=%s", code, pr.out.String())
	}
	if p.resolved.Theme != "auto" {
		t.Fatalf("Theme=%q want auto", p.resolved.Theme)
	}
	// auto keeps delegating to the detector, matching ResolveTheme's contract.
	if got := presentation.ResolveTheme(p.resolved.Theme, nil); got != presentation.ThemeLight {
		t.Fatalf("auto fallback=%q want light", got)
	}
}

func TestBootstrapStructuredOutputStaysANSIFree(t *testing.T) {
	for _, argv := range [][]string{
		{"--output=json", "version"},
		{"--accessible", "version"},
	} {
		env := testkit.CleanEnv(t)
		clearHelpEnv(t)
		writeUserConfig(t, env, `{"ui":{"theme":"dark"}}`)
		pr := newProbedRoot(t)

		if code := runInvocation(context.Background(), pr.root, testBuildInfo(), argv); code != 0 {
			t.Fatalf("argv=%v exit=%d out=%s", argv, code, pr.out.String())
		}
		if out := pr.out.String(); presentation.ContainsCSI([]byte(out)) {
			t.Fatalf("argv=%v produced ANSI with ui.theme=dark: %q", argv, out)
		}
	}
}

// ── §2 one configuration snapshot ───────────────────────────────────────────

func TestBootstrapLoadsConfigOnce(t *testing.T) {
	testkit.CleanEnv(t)
	p := installProbe(t)
	pr := newProbedRoot(t)

	if code := runInvocation(context.Background(), pr.root, testBuildInfo(), []string{"probe"}); code != 0 {
		t.Fatalf("exit=%d out=%s", code, pr.out.String())
	}
	if got := p.configLoads.Load(); got != 1 {
		t.Fatalf("configLoads=%d want 1", got)
	}
}

func TestBootstrapSharesOneSnapshot(t *testing.T) {
	testkit.CleanEnv(t)
	installProbe(t)
	pr := newProbedRoot(t)

	if code := runInvocation(context.Background(), pr.root, testBuildInfo(), []string{"probe"}); code != 0 {
		t.Fatalf("exit=%d out=%s", code, pr.out.String())
	}
	g := ownerFlags(pr.root)
	if g.snapshot == nil {
		t.Fatal("bootstrap retained no snapshot")
	}
	if pr.ac == nil {
		t.Fatal("probe saw no app context")
	}
	if pr.ac.Config != g.snapshot.Config {
		t.Fatal("app.Context must use the bootstrap snapshot's config")
	}
	if pr.ac.CWD != g.snapshot.CWD {
		t.Fatalf("CWD mismatch: app=%q snapshot=%q", pr.ac.CWD, g.snapshot.CWD)
	}
}

func TestBootstrapSnapshotUsesResolvedSpec(t *testing.T) {
	testkit.CleanEnv(t)
	proj := t.TempDir()
	absProj, err := filepath.Abs(proj)
	if err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(proj, "custom.jsonc")
	if err := os.WriteFile(cfg, []byte(`{"offline":false}`), 0o644); err != nil {
		t.Fatal(err)
	}

	p := installProbe(t)
	pr := newProbedRoot(t)
	argv := []string{"--cwd", absProj, "--config", cfg, "--offline", "--prefer-offline", "probe"}
	if code := runInvocation(context.Background(), pr.root, testBuildInfo(), argv); code != 0 {
		t.Fatalf("exit=%d out=%s", code, pr.out.String())
	}
	if got := p.configLoads.Load(); got != 1 {
		t.Fatalf("configLoads=%d want 1", got)
	}
	got := p.loadOpts[0]
	if got.CWD != absProj || got.ConfigPath != cfg || !got.Offline || !got.PreferOffline {
		t.Fatalf("load options did not carry the invocation flags: %+v", got)
	}
	if pr.ac.CWD != absProj {
		t.Fatalf("app CWD=%q want %q", pr.ac.CWD, absProj)
	}
}

func TestBootstrapDirectDispatchLoadsConfigOnce(t *testing.T) {
	testkit.CleanEnv(t)
	p := installProbe(t)
	pr := newProbedRoot(t)

	// Unknown selector still runs the full direct-dispatch path.
	_ = runInvocation(context.Background(), pr.root, testBuildInfo(), []string{"definitely-not-a-command"})
	if got := p.configLoads.Load(); got != 1 {
		t.Fatalf("configLoads=%d want 1 for direct dispatch", got)
	}
	if got := p.controllers.Load(); got != 1 {
		t.Fatalf("controllers=%d want 1 for direct dispatch", got)
	}
	if got := p.closes.Load(); got != 1 {
		t.Fatalf("closes=%d want 1 for direct dispatch", got)
	}
}

func TestBootstrapMXDispatchLoadsConfigOnce(t *testing.T) {
	testkit.CleanEnv(t)
	p := installProbe(t)
	root := NewMXRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	// Offline so the DLX attempt fails fast without touching the network.
	_ = runInvocation(context.Background(), root, testBuildInfo(), []string{"--offline", "definitely-not-a-package"})
	if got := p.configLoads.Load(); got != 1 {
		t.Fatalf("configLoads=%d want 1 for mx dispatch", got)
	}
	if got := p.controllers.Load(); got != 1 {
		t.Fatalf("controllers=%d want 1 for mx dispatch", got)
	}
	if got := p.closes.Load(); got != 1 {
		t.Fatalf("closes=%d want 1 for mx dispatch", got)
	}
}

// ── §4 repair and informational commands ────────────────────────────────────

func TestBootstrapRepairCommandsSurviveMalformedConfig(t *testing.T) {
	for _, argv := range [][]string{
		{"config", "path"},
		{"config", "validate"},
		{"version"},
		{"--help"},
		{"completion", "bash"},
	} {
		t.Run(strings.Join(argv, "_"), func(t *testing.T) {
			env := testkit.CleanEnv(t)
			writeUserConfig(t, env, `{ this is not valid jsonc `)
			p := installProbe(t)
			pr := newProbedRoot(t)

			code := runInvocation(context.Background(), pr.root, testBuildInfo(), argv)
			// config validate reports the malformed file as its own result; the
			// requirement is that bootstrap does not block the command.
			if code != 0 && argv[len(argv)-1] != "validate" {
				t.Fatalf("argv=%v exit=%d out=%s", argv, code, pr.out.String())
			}
			if got := p.controllers.Load(); got != 1 {
				t.Fatalf("controllers=%d want 1", got)
			}
			if g := ownerFlags(pr.root); g.theme != "auto" {
				t.Fatalf("theme=%q want auto fallback on malformed config", g.theme)
			}
		})
	}
}

func TestBootstrapNormalCommandFailsClosedOnMalformedConfig(t *testing.T) {
	env := testkit.CleanEnv(t)
	writeUserConfig(t, env, `{ this is not valid jsonc `)
	installProbe(t)
	pr := newProbedRoot(t)

	if code := runInvocation(context.Background(), pr.root, testBuildInfo(), []string{"probe"}); code == 0 {
		t.Fatalf("normal command must fail closed on malformed config, out=%s", pr.out.String())
	}
	if pr.ac != nil {
		t.Fatal("command body must not run with malformed config")
	}
}

// ── §5 and §6 controller lifecycle ──────────────────────────────────────────

func TestBootstrapControllerLifecycle(t *testing.T) {
	cases := []struct {
		name     string
		argv     []string
		ctx      func(t *testing.T) context.Context
		wantExit int
	}{
		{name: "success", argv: []string{"probe"}, wantExit: 0},
		{name: "command-error", argv: []string{"probefail"}, wantExit: 1},
		{name: "panic", argv: []string{"probepanic"}, wantExit: apperr.ExitCode(apperr.New(apperr.InternalPanic, "", "", ""))},
		{name: "cancelled", argv: []string{"probe"}, wantExit: 130, ctx: func(t *testing.T) context.Context {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testkit.CleanEnv(t)
			p := installProbe(t)
			pr := newProbedRoot(t)
			if tc.name == "cancelled" {
				// A command that surfaces the cancelled context.
				pr.root.AddCommand(&cobra.Command{
					Use:  "probe-cancel",
					RunE: func(cmd *cobra.Command, args []string) error { return cmd.Context().Err() },
				})
				tc.argv = []string{"probe-cancel"}
			}

			ctx := context.Background()
			if tc.ctx != nil {
				ctx = tc.ctx(t)
			}
			code := runInvocation(ctx, pr.root, testBuildInfo(), tc.argv)
			if code != tc.wantExit {
				t.Fatalf("exit=%d want %d out=%s", code, tc.wantExit, pr.out.String())
			}
			if got := p.controllers.Load(); got != 1 {
				t.Fatalf("controllers=%d want exactly 1", got)
			}
			if got := p.closes.Load(); got != 1 {
				t.Fatalf("closes=%d want exactly 1", got)
			}
		})
	}
}

func TestBootstrapPanicReportedBeforeClose(t *testing.T) {
	testkit.CleanEnv(t)
	p := installProbe(t)
	pr := newProbedRoot(t)

	code := runInvocation(context.Background(), pr.root, testBuildInfo(), []string{"probepanic"})
	if code == 0 {
		t.Fatal("panic must not exit 0")
	}
	out := pr.out.String()
	if !strings.Contains(out, "crash-") {
		t.Fatalf("panic report must carry a crash id, got %q", out)
	}
	if got := p.closes.Load(); got != 1 {
		t.Fatalf("closes=%d want 1 after panic", got)
	}
}
