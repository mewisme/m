package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/presentation"
)

// TestHelpViaExecutePath covers Phase A direct dispatch: "help" must fall through
// to Cobra/topic routing (root.Execute alone does not exercise tryDirectDispatch).
func TestHelpViaExecutePath(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want string
	}{
		{"root", []string{"help"}, "Use \"m help <topic>\""},
		{"errors-index", []string{"help", "--pager=never", "errors"}, "Error help index"},
		{"errors-code", []string{"help", "--pager=never", "errors", "ERR_M_LOCKFILE"}, "ERR_M_LOCKFILE"},
		{"command", []string{"help", "install"}, "Examples:"},
		{"flag-help", []string{"--help"}, "Use \"m help <topic>\""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := testBuildInfo()
			root := NewMRoot(info)
			buf := new(strings.Builder)
			root.SetOut(buf)
			root.SetErr(buf)
			code := execute(root, info, append([]string(nil), tc.argv...))
			out := buf.String()
			if code != 0 {
				t.Fatalf("exit %d\nout:\n%s", code, out)
			}
			if strings.Contains(out, `unknown command "help"`) {
				t.Fatalf("direct dispatch stole help:\n%s", out)
			}
			if !strings.Contains(out, tc.want) {
				t.Fatalf("missing %q in:\n%s", tc.want, out)
			}
		})
	}
}

func TestHelpTopicRunner(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	buf := new(strings.Builder)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"help", "--pager=never", "runner"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "m run") {
		t.Fatalf("topic help missing content:\n%s", out)
	}
}

func TestTopicHelpUsePlainColorAlwaysForcesRich(t *testing.T) {
	caps := presentation.Capabilities{
		StdoutTTY:    false,
		StderrTTY:    false,
		CI:           false,
		DumbTerminal: false,
		NoColorEnv:   true,
		ColorProfile: presentation.ColorProfileASCII,
	}
	opts := presentation.ResolvedOptions{
		RequestedOutput: presentation.OutputAuto,
		EffectiveOutput: presentation.OutputPlain,
		Color:           presentation.TriAlways,
	}
	eff := presentation.Effective(opts, caps)
	if topicHelpUsePlain(opts, caps, eff) {
		t.Fatal("expected Glamour path for --color=always on non-TTY")
	}
}

func TestTopicHelpUsePlainAutoNonTTYStaysPlain(t *testing.T) {
	caps := presentation.Capabilities{
		StdoutTTY:    false,
		StderrTTY:    false,
		ColorProfile: presentation.ColorProfileTrueColor,
	}
	opts := presentation.ResolvedOptions{
		RequestedOutput: presentation.OutputAuto,
		EffectiveOutput: presentation.OutputPlain,
		Color:           presentation.TriAuto,
	}
	eff := presentation.Effective(opts, caps)
	if !topicHelpUsePlain(opts, caps, eff) {
		t.Fatal("expected plain path for auto color on non-TTY")
	}
}

func TestHelpTopicColorAlwaysForcesGlamour(t *testing.T) {
	// strings.Builder stdout is non-TTY; --color=always must still select Glamour.
	t.Setenv("CI", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("NO_COLOR", "")
	root := NewMRoot(testBuildInfo())
	buf := new(strings.Builder)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--color=always", "help", "--pager=never", "runner"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "# Runner") {
		t.Fatalf("plain renderer still selected (# heading):\n%s", out)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI from Glamour with --color=always:\n%q", out)
	}
	if !strings.Contains(out, "•") {
		t.Fatalf("expected Glamour bullet:\n%s", out)
	}
}

func TestHelpTopicConfigColorAlwaysForcesGlamour(t *testing.T) {
	// Regression: --color default must not be literal "auto", or ui.color never applies.
	t.Setenv("CI", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("NO_COLOR", "")
	t.Setenv("MEW_COLOR", "")
	dir := t.TempDir()
	cfgPath := dir + string(os.PathSeparator) + "color-always.jsonc"
	if err := os.WriteFile(cfgPath, []byte(`{"ui":{"color":"always"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := NewMRoot(testBuildInfo())
	buf := new(strings.Builder)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "help", "--pager=never", "runner"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "# Runner") {
		t.Fatalf("plain renderer still selected with ui.color=always:\n%s", out)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI from Glamour with ui.color=always:\n%q", out)
	}
}

func TestHelpCommandWinsOverTopicAlias(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	buf := new(strings.Builder)
	root.SetOut(buf)
	root.SetErr(buf)
	// "trust" is both a command and a lifecycle-trust alias; command wins.
	root.SetArgs([]string{"help", "trust"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Trust lifecycle scripts") {
		t.Fatalf("expected trust command help:\n%s", out)
	}
	if strings.Contains(out, "lifecycle.script_trust") {
		t.Fatalf("got topic content instead of command help:\n%s", out)
	}
}

func TestHelpInstallStillCommand(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	buf := new(strings.Builder)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"help", "install"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Examples:") {
		t.Fatalf("install help missing examples:\n%s", out)
	}
}

func TestHelpErrorsCode(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	buf := new(strings.Builder)
	root.SetOut(buf)
	root.SetErr(buf)
	// Local flags must precede args for Cobra.
	root.SetArgs([]string{"help", "--pager=never", "errors", "ERR_M_LOCKFILE"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "ERR_M_LOCKFILE") {
		t.Fatalf("missing lockfile topic:\n%s", buf.String())
	}
}

func TestHelpUnknownTopic(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	buf := new(strings.Builder)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"help", "no-such-topic-xyz"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error")
	}
}
