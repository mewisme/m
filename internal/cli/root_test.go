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
	if !strings.Contains(out, "check=go") {
		t.Fatalf("doctor output missing go check:\n%s", out)
	}
	if !strings.Contains(out, "doctor=stub") {
		t.Fatalf("doctor output missing stub marker:\n%s", out)
	}
}

func TestMFeaturesJSON(t *testing.T) {
	root := NewMRoot(BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	root.SetArgs([]string{"features", "--format", "json", "--module", "runner", "--status", "planned"})
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
	if !strings.Contains(buf.String(), "Mewx") {
		t.Fatalf("help missing Mewx:\n%s", buf.String())
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
	if got := strings.TrimSpace(buf.String()); got != "m 1.2.3 (abc)" {
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
	if !strings.Contains(out, "registry=") {
		t.Fatalf("missing registry:\n%s", out)
	}
	if !strings.Contains(out, "source=project") {
		t.Fatalf("missing project source:\n%s", out)
	}
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
