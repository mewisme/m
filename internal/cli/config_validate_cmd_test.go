package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/testkit"
)

// validateRun drives `m config validate` through the production invocation path
// so the assertion covers the real exit code, not just the RunE return.
type validateRun struct {
	exit int
	out  string
	err  string
}

func runConfigValidate(t *testing.T, argv ...string) validateRun {
	t.Helper()
	root := NewMRoot(testBuildInfo())
	out, errBuf := new(bytes.Buffer), new(bytes.Buffer)
	root.SetOut(out)
	root.SetErr(errBuf)
	exit := runInvocation(context.Background(), root, testBuildInfo(),
		append([]string{"config", "validate"}, argv...))
	return validateRun{exit: exit, out: out.String(), err: errBuf.String()}
}

// writeProject creates a project root with an m.jsonc holding body.
func writeProject(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if body != "" {
		if err := os.WriteFile(filepath.Join(dir, "m.jsonc"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// ── §8 human-mode exit behavior ─────────────────────────────────────────────

func TestConfigValidateHumanValidExitsZero(t *testing.T) {
	env := testkit.CleanEnv(t)
	writeUserConfig(t, env, `{"offline": true, "install": {"linker": "hoisted"}}`)

	got := runConfigValidate(t)
	if got.exit != 0 {
		t.Fatalf("exit=%d want 0\nstdout:%s\nstderr:%s", got.exit, got.out, got.err)
	}
	if !strings.Contains(got.out, "is valid") {
		t.Errorf("stdout should confirm validity:\n%s", got.out)
	}
}

// TestConfigValidateHumanInvalidExitsNonZero is the headline fix: the old
// command printed the failure and still returned nil.
func TestConfigValidateHumanInvalidExitsNonZero(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string // substring the report must contain
	}{
		{"unknown key", `{"install": {"linkr": "hoisted"}}`, "install.linkr"},
		{"bad enum", `{"install": {"linker": "magic"}}`, "install.linker"},
		{"bad type", `{"offline": "yes"}`, "offline"},
		{"below minimum", `{"transaction": {"snapshot_retention": -1}}`, "snapshot_retention"},
		{"malformed jsonc", `{ this is not valid `, "JSONC"},
		{"root not an object", `[1, 2, 3]`, "object"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := testkit.CleanEnv(t)
			writeUserConfig(t, env, tc.body)

			got := runConfigValidate(t)
			if got.exit == 0 {
				t.Fatalf("invalid config exited 0\nstdout:%s\nstderr:%s", got.out, got.err)
			}
			// The complete report is still printed, not replaced by the error.
			if !strings.Contains(got.out, "is invalid") {
				t.Errorf("stdout should carry the report:\n%s", got.out)
			}
			if !strings.Contains(got.out, tc.want) {
				t.Errorf("report should mention %q:\n%s", tc.want, got.out)
			}
		})
	}
}

// TestConfigValidateHumanWarningsExitZero: a legacy spelling still works, so it
// must not fail the command.
func TestConfigValidateHumanWarningsExitZero(t *testing.T) {
	env := testkit.CleanEnv(t)
	writeUserConfig(t, env, `{"prefer-offline": true}`)

	got := runConfigValidate(t)
	if got.exit != 0 {
		t.Fatalf("warning exited %d, want 0\nstdout:%s\nstderr:%s", got.exit, got.out, got.err)
	}
	if !strings.Contains(got.out, "prefer-offline") {
		t.Errorf("warning should name the key the user typed:\n%s", got.out)
	}
	if !strings.Contains(got.out, "prefer_offline") {
		t.Errorf("warning should name the replacement:\n%s", got.out)
	}
}

// TestConfigValidateStrictPromotesWarnings pairs with the test above: the same
// document fails once the user asks for strict.
func TestConfigValidateStrictPromotesWarnings(t *testing.T) {
	env := testkit.CleanEnv(t)
	writeUserConfig(t, env, `{"prefer-offline": true}`)

	got := runConfigValidate(t, "--strict")
	if got.exit == 0 {
		t.Fatalf("--strict exited 0 on a legacy key\nstdout:%s\nstderr:%s", got.out, got.err)
	}
	if !strings.Contains(got.out, "is invalid") {
		t.Errorf("strict report missing:\n%s", got.out)
	}
}

func TestConfigValidateMissingFileIsValid(t *testing.T) {
	testkit.CleanEnv(t) // clean home, no config written

	got := runConfigValidate(t)
	if got.exit != 0 {
		t.Fatalf("absent config exited %d, want 0\nstdout:%s\nstderr:%s", got.exit, got.out, got.err)
	}
	if !strings.Contains(got.out, "is valid") {
		t.Errorf("absent config should be valid:\n%s", got.out)
	}
}

// ── §8/§9 JSON-mode exit behavior and schema ────────────────────────────────

func TestConfigValidateJSONValidExitsZero(t *testing.T) {
	env := testkit.CleanEnv(t)
	writeUserConfig(t, env, `{"offline": true}`)

	got := runConfigValidate(t, "--output", "json")
	if got.exit != 0 {
		t.Fatalf("exit=%d want 0\nstdout:%s\nstderr:%s", got.exit, got.out, got.err)
	}
	doc := decodeValidateJSON(t, got.out)
	if doc["valid"] != true {
		t.Errorf(`"valid" = %v, want true: %s`, doc["valid"], got.out)
	}
	if doc["scope"] != "user" {
		t.Errorf(`"scope" = %v, want user`, doc["scope"])
	}
}

// TestConfigValidateJSONInvalidEmitsReportAndExitsNonZero is the §8 pair: the
// machine report is emitted AND the exit code is non-zero, with exactly one
// document on stdout.
func TestConfigValidateJSONInvalidEmitsReportAndExitsNonZero(t *testing.T) {
	env := testkit.CleanEnv(t)
	writeUserConfig(t, env, `{"install": {"linker": "magic"}}`)

	got := runConfigValidate(t, "--output", "json")
	if got.exit == 0 {
		t.Fatalf("invalid JSON run exited 0\nstdout:%s", got.out)
	}
	doc := decodeValidateJSON(t, got.out)
	if doc["valid"] != false {
		t.Errorf(`"valid" = %v, want false: %s`, doc["valid"], got.out)
	}

	diags, ok := doc["diagnostics"].([]any)
	if !ok || len(diags) == 0 {
		t.Fatalf("diagnostics missing from report: %s", got.out)
	}
	first, _ := diags[0].(map[string]any)
	for _, field := range []string{"severity", "code", "message", "key", "path"} {
		if _, present := first[field]; !present {
			t.Errorf("diagnostic missing %q: %v", field, first)
		}
	}
	if first["key"] != "install.linker" {
		t.Errorf(`diagnostic key = %v, want install.linker`, first["key"])
	}
	if first["severity"] != "error" {
		t.Errorf(`severity = %v, want error`, first["severity"])
	}
	if len(doc["errors"].([]any)) == 0 {
		t.Errorf(`"errors" should be populated: %s`, got.out)
	}
}

// TestConfigValidateJSONEmitsSingleDocument: the reporter must not append its
// own error document after the report.
func TestConfigValidateJSONEmitsSingleDocument(t *testing.T) {
	env := testkit.CleanEnv(t)
	writeUserConfig(t, env, `{"offline": "yes"}`)

	got := runConfigValidate(t, "--output", "json")
	if got.exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	dec := json.NewDecoder(strings.NewReader(got.out))
	var first any
	if err := dec.Decode(&first); err != nil {
		t.Fatalf("first document did not decode: %v\n%s", err, got.out)
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		t.Fatalf("stdout carried a second JSON document:\n%s", got.out)
	}
}

// TestConfigValidateJSONHasNoANSIOrHeadings pins §9: machine output is data
// only.
func TestConfigValidateJSONHasNoANSIOrHeadings(t *testing.T) {
	env := testkit.CleanEnv(t)
	writeUserConfig(t, env, `{"install": {"linker": "magic"}}`)

	got := runConfigValidate(t, "--output", "json")
	if strings.Contains(got.out, "\x1b[") {
		t.Errorf("JSON output carried ANSI escapes:\n%q", got.out)
	}
	for _, heading := range []string{"is invalid", "is valid", "Errors:", "Warnings:", "×", "✓"} {
		if strings.Contains(got.out, heading) {
			t.Errorf("JSON output carried human heading %q:\n%s", heading, got.out)
		}
	}
}

// TestConfigValidateJSONWarningsExitZero: warnings appear in the report but do
// not change the exit status.
func TestConfigValidateJSONWarningsExitZero(t *testing.T) {
	env := testkit.CleanEnv(t)
	writeUserConfig(t, env, `{"prefer-offline": true}`)

	got := runConfigValidate(t, "--output", "json")
	if got.exit != 0 {
		t.Fatalf("warning exited %d, want 0\nstdout:%s", got.exit, got.out)
	}
	doc := decodeValidateJSON(t, got.out)
	if doc["valid"] != true {
		t.Errorf(`"valid" = %v, want true`, doc["valid"])
	}
	warns, _ := doc["warnings"].([]any)
	if len(warns) != 1 {
		t.Fatalf("warnings = %d, want 1: %s", len(warns), got.out)
	}
	w, _ := warns[0].(map[string]any)
	if w["severity"] != "warning" {
		t.Errorf("severity = %v, want warning", w["severity"])
	}
	if w["replacement"] != "prefer_offline" {
		t.Errorf("replacement = %v, want prefer_offline", w["replacement"])
	}
}

func decodeValidateJSON(t *testing.T, out string) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, out)
	}
	return doc
}

// ── §7 scope selection and aggregation ──────────────────────────────────────

// TestConfigValidateEffectiveAggregatesBothFiles: effective scope reports one
// entry per file that exists, in resolution order, and fails when either is bad.
func TestConfigValidateEffectiveAggregatesBothFiles(t *testing.T) {
	env := testkit.CleanEnv(t)
	writeUserConfig(t, env, `{"offline": true}`)
	proj := writeProject(t, `{"install": {"linker": "magic"}}`)

	root := NewMRoot(testBuildInfo())
	out := new(bytes.Buffer)
	root.SetOut(out)
	root.SetErr(out)
	exit := runInvocation(context.Background(), root, testBuildInfo(),
		[]string{"--cwd", proj, "--output", "json", "config", "validate", "--scope", "effective"})

	if exit == 0 {
		t.Fatalf("bad project file exited 0\n%s", out.String())
	}
	doc := decodeValidateJSON(t, out.String())
	files, _ := doc["files"].([]any)
	if len(files) != 2 {
		t.Fatalf("files = %d, want 2 (user then project): %s", len(files), out.String())
	}
	first, _ := files[0].(map[string]any)
	second, _ := files[1].(map[string]any)
	if first["scope"] != "user" || second["scope"] != "project" {
		t.Errorf("scopes out of resolution order: %v then %v", first["scope"], second["scope"])
	}
	if first["valid"] != true {
		t.Errorf("clean user file marked invalid: %v", first)
	}
	if second["valid"] != false {
		t.Errorf("broken project file marked valid: %v", second)
	}
	if doc["keys"].(float64) != 2 {
		t.Errorf("keys = %v, want 2 summed across files", doc["keys"])
	}
}

// TestConfigValidateProjectScopeRejectsUserOnlyKey: the scope check reaches the
// command, and the message names the key and both scopes.
func TestConfigValidateProjectScopeRejectsUserOnlyKey(t *testing.T) {
	testkit.CleanEnv(t)
	proj := writeProject(t, `{"cache": {"dir": "/tmp/cache"}}`)

	root := NewMRoot(testBuildInfo())
	out := new(bytes.Buffer)
	root.SetOut(out)
	root.SetErr(out)
	exit := runInvocation(context.Background(), root, testBuildInfo(),
		[]string{"--cwd", proj, "config", "validate", "--scope", "project"})

	if exit == 0 {
		t.Fatalf("user-only key accepted in project scope\n%s", out.String())
	}
	for _, want := range []string{"cache.dir", "project", "user"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("report should mention %q:\n%s", want, out.String())
		}
	}
}

// TestConfigValidateContinuesPastBrokenUserFile: one broken file must not hide
// problems in the next one.
func TestConfigValidateContinuesPastBrokenUserFile(t *testing.T) {
	env := testkit.CleanEnv(t)
	writeUserConfig(t, env, `{ broken `)
	proj := writeProject(t, `{"install": {"linkr": true}}`)

	root := NewMRoot(testBuildInfo())
	out := new(bytes.Buffer)
	root.SetOut(out)
	root.SetErr(out)
	exit := runInvocation(context.Background(), root, testBuildInfo(),
		[]string{"--cwd", proj, "config", "validate", "--scope", "effective"})

	if exit == 0 {
		t.Fatal("expected non-zero exit")
	}
	// Both problems are reported, from both files.
	if !strings.Contains(out.String(), "JSONC") {
		t.Errorf("user syntax error missing:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "install.linkr") {
		t.Errorf("project error missing; a broken first file hid it:\n%s", out.String())
	}
}

// TestConfigValidateSecretNeverPrinted: §10 — a pasted token must not reach the
// report in either output mode.
func TestConfigValidateSecretNeverPrinted(t *testing.T) {
	const token = "npm_aBcDeF1234567890secret"
	for _, argv := range [][]string{nil, {"--output", "json"}} {
		env := testkit.CleanEnv(t)
		writeUserConfig(t, env, `{"registry": {"auth_token_env": "`+token+`"}}`)

		got := runConfigValidate(t, argv...)
		if got.exit == 0 {
			t.Fatalf("argv=%v inline secret accepted\n%s", argv, got.out)
		}
		if strings.Contains(got.out, token) || strings.Contains(got.err, token) {
			t.Errorf("argv=%v output leaked the secret:\nstdout:%s\nstderr:%s", argv, got.out, got.err)
		}
		if !strings.Contains(got.out, "auth_token_env") {
			t.Errorf("argv=%v report should name the key:\n%s", argv, got.out)
		}
	}
}

// TestConfigValidateInvalidScopeIsUsageError keeps scope validation ahead of any
// file reading.
func TestConfigValidateInvalidScopeIsUsageError(t *testing.T) {
	testkit.CleanEnv(t)
	got := runConfigValidate(t, "--scope", "nonsense")
	if got.exit == 0 {
		t.Fatal("invalid scope exited 0")
	}
}

// ── §6 loader integration ───────────────────────────────────────────────────

// TestNormalCommandFailsClosedOnInvalidValue: the loader shares the validator,
// so a value the validate command rejects also stops an ordinary command. The
// old loader only checked key names, so a bad enum loaded fine.
func TestNormalCommandFailsClosedOnInvalidValue(t *testing.T) {
	cases := map[string]string{
		"bad enum":      `{"install": {"linker": "magic"}}`,
		"bad type":      `{"offline": "yes"}`,
		"below minimum": `{"transaction": {"snapshot_retention": -5}}`,
		"unknown key":   `{"install": {"linkr": "hoisted"}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			env := testkit.CleanEnv(t)
			writeUserConfig(t, env, body)
			pr := newProbedRoot(t)

			if exit := runInvocation(context.Background(), pr.root, testBuildInfo(), []string{"probe"}); exit == 0 {
				t.Fatalf("normal command loaded invalid config\n%s", pr.out.String())
			}
		})
	}
}

// TestConfigValidateRunsOnConfigTheLoaderRejects: validate is a repair command,
// so it must still run when the config it validates cannot load.
func TestConfigValidateRunsOnConfigTheLoaderRejects(t *testing.T) {
	env := testkit.CleanEnv(t)
	writeUserConfig(t, env, `{"install": {"linker": "magic"}}`)

	got := runConfigValidate(t)
	if got.exit == 0 {
		t.Fatal("expected non-zero exit for invalid config")
	}
	// The report is the command's own output, not a bootstrap failure.
	if !strings.Contains(got.out, "install.linker") {
		t.Errorf("validate did not produce its own report:\n%s", got.out)
	}
}

// TestSuppressReportKeepsTypingAndExit: the suppression wrapper must not change
// how an error is classified. If it did, suppressing the duplicate JSON document
// would also change the exit code the caller sees.
func TestSuppressReportKeepsTypingAndExit(t *testing.T) {
	inner := apperr.New(apperr.Config, "config.validate", "config.jsonc:offline", "invalid configuration")
	wrapped := suppressReport(inner)

	if !reportSuppressed(wrapped) {
		t.Error("wrapped error not recognized as suppressed")
	}
	if reportSuppressed(inner) {
		t.Error("unwrapped error reported as suppressed")
	}
	if got, want := apperr.CodeOf(wrapped), apperr.CodeOf(inner); got != want {
		t.Errorf("code changed by wrapping: %s vs %s", got, want)
	}
	if got, want := apperr.ExitCode(wrapped), apperr.ExitCode(inner); got != want {
		t.Errorf("exit code changed by wrapping: %d vs %d", got, want)
	}
	if suppressReport(nil) != nil {
		t.Error("suppressReport(nil) should stay nil")
	}
	// classifyCLIError must pass a suppressed typed error through unchanged, or
	// the wrapper would be stripped before runInvocation checks it.
	if !reportSuppressed(classifyCLIError(wrapped)) {
		t.Error("classifyCLIError dropped the suppression marker")
	}
}
