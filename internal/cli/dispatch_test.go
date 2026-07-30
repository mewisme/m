package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/jsonfile"
)

func TestParsePhaseAGlobalsAndForwarding(t *testing.T) {
	phase, err := ParsePhaseA([]string{"--cwd", "./app", "build", "--mode", "production"})
	if err != nil {
		t.Fatal(err)
	}
	if phase.Selector != "build" {
		t.Fatalf("selector=%q", phase.Selector)
	}
	if phase.Leading.cwd != "./app" {
		t.Fatalf("cwd=%q", phase.Leading.cwd)
	}
	if got := strings.Join(phase.ForwardedArgs, ","); got != "--mode,production" {
		t.Fatalf("forwarded=%v", phase.ForwardedArgs)
	}

	phase, err = ParsePhaseA([]string{"--cwd=./app", "build", "--", "--mode", "production"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(phase.ForwardedArgs, ","); got != "--mode,production" {
		t.Fatalf("forwarded=%v", phase.ForwardedArgs)
	}

	phase, err = ParsePhaseA([]string{"-r", "--workspace-concurrency", "2", "build"})
	if err != nil {
		t.Fatal(err)
	}
	if !phase.Leading.recursive || phase.Leading.wsConcurrency != 2 {
		t.Fatalf("leading=%+v", phase.Leading)
	}
	if phase.Selector != "build" {
		t.Fatalf("selector=%q", phase.Selector)
	}

	phase, err = ParsePhaseA([]string{"-r", "build", "--workspace-concurrency", "2"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(phase.ForwardedArgs, ","); got != "--workspace-concurrency,2" {
		t.Fatalf("forwarded=%v", phase.ForwardedArgs)
	}
}

func TestParsePhaseAUnknownFlag(t *testing.T) {
	_, err := ParsePhaseA([]string{"--not-a-mew-flag", "dev"})
	if err == nil || apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("err=%v", err)
	}
}

func TestForwardedScriptArgsParity(t *testing.T) {
	args := []string{"build", "--mode", "production"}
	if got := forwardedScriptArgs(args, -1); strings.Join(got, ",") != "--mode,production" {
		t.Fatalf("got=%v", got)
	}
	args = []string{"build", "--", "--mode", "production"}
	if got := forwardedScriptArgs(args, 1); strings.Join(got, ",") != "--mode,production" {
		t.Fatalf("got=%v", got)
	}
}

func TestDirectScriptsGate(t *testing.T) {
	t.Setenv("MEW_EXPERIMENTAL_DIRECT_SCRIPTS", "")
	eff := &config.Effective{Values: map[string]config.Value{}}
	if DirectScriptsEnabled(eff) {
		t.Fatal("expected disabled")
	}
	t.Setenv("MEW_EXPERIMENTAL_DIRECT_SCRIPTS", "1")
	if !DirectScriptsEnabled(eff) {
		t.Fatal("expected env enabled")
	}
	t.Setenv("MEW_EXPERIMENTAL_DIRECT_SCRIPTS", "")
	eff.Values["runner.direct_scripts.enabled"] = config.Value{Raw: true, Source: config.SourceProject}
	if !DirectScriptsEnabled(eff) {
		t.Fatal("expected config enabled")
	}
}

func TestReservedDriftAgainstCobra(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	if missing := driftAgainstShippedBuiltins(root); len(missing) > 0 {
		t.Fatalf("cobra tree missing shipped names: %v", missing)
	}
}

func TestLeadingGlobalParserDrift(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	got := phaseAParserFlagNames(root)
	for _, name := range rootPersistentFlagNames(root) {
		if !containsString(got, name) {
			t.Fatalf("phase A parser missing root flag %q in %v", name, got)
		}
	}
}

func TestBuiltinBeatsScript(t *testing.T) {
	projDir := t.TempDir()
	pkg := map[string]any{
		"name":    "demo",
		"version": "1.0.0",
		"scripts": map[string]string{"install": "echo nope"},
	}
	raw, _ := jsonfile.Marshal(pkg)
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	root := NewMRoot(testBuildInfo())
	t.Setenv("MEW_EXPERIMENTAL_DIRECT_SCRIPTS", "1")
	phase, err := ParsePhaseA([]string{"install"})
	if err != nil {
		t.Fatal(err)
	}
	res := ResolveDispatch(root, phase, projDir, &config.Effective{Values: map[string]config.Value{
		"runner.direct_scripts.enabled": {Raw: true},
	}})
	if res.Kind != OutcomeBuiltin {
		t.Fatalf("kind=%s", res.Kind)
	}
}

func TestExactScriptCaseSensitive(t *testing.T) {
	projDir := t.TempDir()
	pkg := map[string]any{
		"name":    "demo",
		"version": "1.0.0",
		"scripts": map[string]string{"dev": "echo dev"},
	}
	raw, _ := jsonfile.Marshal(pkg)
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	root := NewMRoot(testBuildInfo())
	eff := &config.Effective{Values: map[string]config.Value{
		"runner.direct_scripts.enabled": {Raw: true},
	}}
	phase := PhaseAResult{Selector: "Dev"}
	res := ResolveDispatch(root, phase, projDir, eff)
	if res.Kind == OutcomeScript {
		t.Fatal("case mismatch must not execute")
	}
	phase.Selector = "dev"
	res = ResolveDispatch(root, phase, projDir, eff)
	if res.Kind != OutcomeScript {
		t.Fatalf("kind=%s", res.Kind)
	}
}

func TestGateOffExactScriptMessage(t *testing.T) {
	t.Setenv("MEW_EXPERIMENTAL_DIRECT_SCRIPTS", "")
	projDir := t.TempDir()
	pkg := map[string]any{
		"name":    "demo",
		"version": "1.0.0",
		"scripts": map[string]string{"dev": "echo dev"},
	}
	raw, _ := jsonfile.Marshal(pkg)
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	root := NewMRoot(testBuildInfo())
	phase := PhaseAResult{Selector: "dev"}
	res := ResolveDispatch(root, phase, projDir, &config.Effective{Values: map[string]config.Value{}})
	if res.Kind != OutcomeSuggest {
		t.Fatalf("kind=%s", res.Kind)
	}
	if res.Err == nil || !strings.Contains(res.Err.Error(), "Direct script shortcuts are disabled") {
		t.Fatalf("err=%v", res.Err)
	}
	if !strings.Contains(res.Err.Error(), "m run dev") {
		t.Fatalf("err=%v", res.Err)
	}
}

func TestBuiltinTypoOutsideProject(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	phase := PhaseAResult{Selector: "instal"}
	res := ResolveDispatch(root, phase, t.TempDir(), nil)
	if res.Kind != OutcomeSuggest {
		t.Fatalf("kind=%s", res.Kind)
	}
	foundInstall := false
	for _, s := range res.Suggestions {
		if s.Name == "install" {
			foundInstall = true
		}
	}
	if !foundInstall {
		t.Fatalf("suggestions=%v", res.Suggestions)
	}
}

func TestSuggestionRankingAndLimit(t *testing.T) {
	candidates := [][]Suggestion{
		{{Name: "dev", Kind: DispatchScript, Invocation: "m run dev", Distance: 1}},
		{{Name: "install", Kind: DispatchBuiltin, Invocation: "m install", Distance: 2}},
		{{Name: "i", Kind: DispatchAlias, Invocation: "m i", Distance: 2}},
	}
	got := mergeSuggestions(candidates...)
	if len(got) > 3 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Kind != DispatchBuiltin {
		t.Fatalf("first=%+v", got[0])
	}
}

func TestDispatchJSONBuiltin(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	res := ResolveDispatch(root, PhaseAResult{Selector: "install"}, "", nil)
	raw, err := encodeDispatchJSON(res, "install")
	if err != nil {
		t.Fatal(err)
	}
	var doc dispatchJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != 1 || doc.Kind != "builtin" || doc.Path != "install" {
		t.Fatalf("%+v", doc)
	}
}

func TestIsPlausibleScriptSelector(t *testing.T) {
	if !isPlausibleScriptSelector("dev") {
		t.Fatal("dev should be plausible")
	}
	if isPlausibleScriptSelector("not-a-command") {
		t.Fatal("arbitrary phrase should not be plausible")
	}
}

func TestReservedScriptSuggestionUsesRun(t *testing.T) {
	s := formatScriptInvocation("add", true, true)
	if s != "m run add" {
		t.Fatalf("got %q", s)
	}
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
