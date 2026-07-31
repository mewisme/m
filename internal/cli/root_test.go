package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
)

func TestMRootHelp(t *testing.T) {
	root := NewMRoot(BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"features", "version", "development", "config", "Mew"} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q:\n%s", want, out)
		}
	}
}

func TestDevelopmentDoctor(t *testing.T) {
	root := NewMRoot(BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"development", "doctor"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Development prerequisites checked") {
		t.Fatalf("doctor output missing title:\n%s", out)
	}
	if !strings.Contains(out, "Go") {
		t.Fatalf("doctor output missing Go check:\n%s", out)
	}
	if !strings.Contains(out, "(stub)") {
		t.Fatalf("doctor output missing stub marker:\n%s", out)
	}
}

func TestMFeaturesJSON(t *testing.T) {
	root := NewMRoot(BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	root.SetArgs([]string{"features", "--format", "json", "--module", "runner", "--status", "shipped"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"runner.direct-shortcuts"`) {
		t.Fatalf("unexpected output:\n%s", out)
	}
	if strings.Contains(out, `"tests"`) {
		t.Fatal("user-facing JSON must not include tests field")
	}
}

func TestMXRootHelp(t *testing.T) {
	root := NewMXRoot(BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "MewJS") {
		t.Fatalf("help missing MewJS:\n%s", buf.String())
	}
}

func TestVersionSubcommand(t *testing.T) {
	root := NewMRoot(BuildInfo{Version: "1.2.3", Commit: "abc"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "m 1.2.3") || !strings.Contains(got, "abc") {
		t.Fatalf("version output = %q", got)
	}
}

func TestConfigListSources(t *testing.T) {
	root := NewMRoot(BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	modRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(modRoot, "fixtures", "identity", "mew-native")
	root.SetArgs([]string{"--cwd", fixture, "config", "list", "--sources"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "registry") {
		t.Fatalf("missing registry:\n%s", out)
	}
	if !strings.Contains(out, "project") {
		t.Fatalf("missing project source:\n%s", out)
	}
}

func TestConfigListValuesColumn(t *testing.T) {
	root := NewMRoot(BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	modRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(modRoot, "fixtures", "identity", "mew-native")
	root.SetArgs([]string{"--cwd", fixture, "--no-color", "config", "list"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "VALUES") {
		t.Fatalf("missing VALUES header:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "log.level") {
			continue
		}
		if strings.Contains(line, "error") && strings.Contains(line, "warn") && strings.Contains(line, "info") && strings.Contains(line, "debug") {
			return
		}
		t.Fatalf("log.level row missing error|warn|info|debug:\n%s\nfull:\n%s", line, out)
	}
	t.Fatalf("log.level row missing:\n%s", out)
}

func TestRecoverPanic(t *testing.T) {
	var errW bytes.Buffer
	rep := diagnostics.NewReporter(diagnostics.Options{
		Out: ioDiscard{}, Err: &errW, Format: "silent", Color: diagnostics.ColorNever,
	})
	code := RecoverPanic(rep, func() { panic("boom") })
	if code != 1 {
		t.Fatalf("exit=%d", code)
	}
	if !strings.Contains(errW.String(), "ERR_M_INTERNAL_PANIC") {
		t.Fatalf("output=%q", errW.String())
	}
}

func TestClassifyUnknownCommandAsUsage(t *testing.T) {
	root := NewMRoot(BuildInfo{Version: "0.0.0-test"})
	var errW bytes.Buffer
	root.SetOut(ioDiscard{})
	root.SetErr(&errW)
	root.SetArgs([]string{"definitely-not-a-command"})
	err := root.Execute()
	if err == nil {
		t.Fatal(err)
	}
	got := classifyCLIError(err)
	if apperr.CodeOf(got) != apperr.Usage {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(got), got)
	}
	if apperr.ExitCode(got) != 2 {
		t.Fatalf("exit=%d", apperr.ExitCode(got))
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
