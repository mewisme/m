package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompletionNoANSI(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			buf := new(strings.Builder)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs([]string{"completion", shell})
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			out := buf.String()
			if strings.Contains(out, "\x1b") {
				t.Fatalf("completion %s contains ANSI escapes", shell)
			}
			if strings.Contains(out, "lipgloss") {
				t.Fatalf("completion %s contains lipgloss sequences", shell)
			}
		})
	}
}

func TestInstallHelpExamples(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	buf := new(strings.Builder)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"install", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Examples:") {
		t.Fatalf("install help missing Examples:\n%s", out)
	}
}

// Regenerate help goldens when templates change: go test ./internal/cli -run TestWriteHelpGoldens -count=1
func TestWriteHelpGoldens(t *testing.T) {
	if os.Getenv("MEW_UPDATE_HELP_GOLDEN") == "" {
		t.Skip("set MEW_UPDATE_HELP_GOLDEN=1 to rewrite help goldens")
	}
	cases := []struct {
		name string
		root *cobra.Command
		file string
	}{
		{"m", NewMRoot(testBuildInfo()), "m-root.txt"},
		{"mx", NewMXRoot(testBuildInfo()), "mx-root.txt"},
	}
	dir := filepath.Join("..", "..", "testdata", "cli", "help-golden")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder
			tc.root.SetOut(&buf)
			tc.root.SetErr(&buf)
			tc.root.SetArgs([]string{"--help"})
			if err := tc.root.Execute(); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, tc.file)
			if err := os.WriteFile(path, []byte(normalizeEOL(buf.String())), 0o644); err != nil {
				t.Fatal(err)
			}
		})
	}
}
