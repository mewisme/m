package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/diagnostics"
)

func testBuildInfo() BuildInfo {
	return BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"}
}

func normalizeEOL(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func TestHelpGoldens(t *testing.T) {
	cases := []struct {
		name string
		root *cobra.Command
		file string
	}{
		{"m", NewMRoot(testBuildInfo()), "m-root.txt"},
		{"mx", NewMXRoot(testBuildInfo()), "mx-root.txt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			tc.root.SetOut(buf)
			tc.root.SetErr(buf)
			tc.root.SetArgs([]string{"--help"})
			if err := tc.root.Execute(); err != nil {
				t.Fatal(err)
			}
			got := normalizeEOL(buf.String())
			path := filepath.Join("..", "..", "testdata", "cli", "help-golden", tc.file)
			wantBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			want := normalizeEOL(string(wantBytes))
			if got != want {
				t.Fatalf("help golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", tc.file, got, want)
			}
		})
	}
}

func TestVersionJSONAndBuildDate(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"version", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var doc map[string]string
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"binary", "version", "commit", "buildDate"} {
		if _, ok := doc[k]; !ok {
			t.Fatalf("missing key %q in %v", k, doc)
		}
	}
	if doc["version"] != "0.0.0-test" || doc["commit"] != "deadbeef" || doc["buildDate"] != "2026-01-01" {
		t.Fatalf("%v", doc)
	}
	if doc["binary"] != "m" {
		t.Fatalf("binary=%q", doc["binary"])
	}

	buf.Reset()
	root = NewMRoot(testBuildInfo())
	root.SetOut(buf)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	out := normalizeEOL(buf.String())
	if !strings.Contains(out, "m 0.0.0-test (deadbeef)") {
		t.Fatalf("version text:\n%s", out)
	}
	if !strings.Contains(out, "built 2026-01-01") {
		t.Fatalf("missing build date:\n%s", out)
	}
}

func TestReservedNamesAndStub(t *testing.T) {
	names := ReservedNames()
	for _, want := range []string{"install", "i", "run", "version", "completion", "__dispatch"} {
		if !IsReserved(want) {
			t.Fatalf("expected reserved %q in %v", want, names)
		}
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Fatalf("ReservedNames not sorted: %v", names)
		}
	}

	root := NewMRoot(testBuildInfo())
	var errW bytes.Buffer
	root.SetOut(ioDiscard{})
	root.SetErr(ioDiscard{})
	rep := diagnostics.NewReporter(diagnostics.Options{
		Out: ioDiscard{}, Err: &errW, Format: "silent", Color: diagnostics.ColorNever,
	})
	_ = rep
	root.SetArgs([]string{"plan"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected unimplemented")
	}
	if apperr.CodeOf(err) != apperr.Unimplemented {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	if apperr.ExitCode(err) != 1 {
		t.Fatalf("exit=%d", apperr.ExitCode(err))
	}
	if !strings.Contains(err.Error(), "0028") {
		t.Fatalf("err=%v", err)
	}
}

func TestDispatchInstall(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"__dispatch", "install"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(buf.String())
	if got != "kind=builtin path=install" {
		t.Fatalf("%q", got)
	}
}

func TestCWDSetsAppContext(t *testing.T) {
	modRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(modRoot, "fixtures", "projects", "empty-package-json")
	abs, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	var gotCWD string
	root.AddCommand(&cobra.Command{
		Use: "print-cwd",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "cli", "print-cwd", "missing app context")
			}
			gotCWD = ac.CWD
			return nil
		},
	})
	root.SetOut(ioDiscard{})
	root.SetErr(ioDiscard{})
	root.SetArgs([]string{"--cwd", fixture, "print-cwd"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if gotCWD != abs {
		t.Fatalf("CWD=%q want %q", gotCWD, abs)
	}
}

func TestCancelledContextExit130(t *testing.T) {
	root := NewMRoot(testBuildInfo())
	root.AddCommand(&cobra.Command{
		Use: "waitcancel",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Context().Err()
		},
	})
	root.SetOut(ioDiscard{})
	root.SetErr(ioDiscard{})
	root.SetArgs([]string{"waitcancel"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	code := ExecuteWithContext(root, ctx)
	if code != 130 {
		t.Fatalf("exit=%d want 130", code)
	}
}

func TestInvokedBinaryDisplayName(t *testing.T) {
	cases := []struct {
		argv0, fallback, wantInv, wantDisp string
	}{
		{"m", "m", "m", "m"},
		{"mew.exe", "m", "mew", "mew"},
		{"/usr/bin/mewx", "mx", "mewx", "mewx"},
		{"something-else", "m", "m", "m"},
		{"mx", "mx", "mx", "mx"},
	}
	for _, tc := range cases {
		if InvokedBinary(tc.argv0, tc.fallback) != tc.wantInv {
			t.Errorf("InvokedBinary(%q)=%q", tc.argv0, InvokedBinary(tc.argv0, tc.fallback))
		}
		if DisplayName(tc.wantInv) != tc.wantDisp {
			t.Errorf("DisplayName(%q)=%q", tc.wantInv, DisplayName(tc.wantInv))
		}
	}
}

func TestFlagsMatrix(t *testing.T) {
	modRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(modRoot, "testdata", "cli", "flags-matrix.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name           string   `json:"name"`
		Args           []string `json:"args"`
		Exit           int      `json:"exit"`
		StdoutContains []string `json:"stdoutContains"`
		StderrContains []string `json:"stderrContains"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}

	info := testBuildInfo()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			root := NewMRoot(info)
			var outB, errB bytes.Buffer
			root.SetOut(&outB)
			root.SetErr(&errB)
			root.SetArgs(append([]string{}, tc.Args...))

			// Route diagnostics to errB for exit-path cases.
			code := 0
			err := root.Execute()
			if err != nil {
				classified := classifyCLIError(err)
				rep := diagnostics.NewReporter(diagnostics.Options{
					Out: &outB, Err: &errB, Format: "default", Color: diagnostics.ColorNever,
				})
				rep.Error(classified)
				code = apperr.ExitCode(classified)
			}
			if code != tc.Exit {
				t.Fatalf("exit=%d want %d err=%v\nstdout=%s\nstderr=%s", code, tc.Exit, err, outB.String(), errB.String())
			}
			stdout := outB.String()
			stderr := errB.String()
			for _, want := range tc.StdoutContains {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout missing %q:\n%s", want, stdout)
				}
			}
			for _, want := range tc.StderrContains {
				if !strings.Contains(stderr, want) {
					t.Fatalf("stderr missing %q:\n%s", want, stderr)
				}
			}
		})
	}
}

func TestCompletionShells(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			buf := new(bytes.Buffer)
			root := NewMRoot(testBuildInfo())
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs([]string{"completion", shell})
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if buf.Len() < 20 {
				t.Fatalf("short completion for %s: %q", shell, buf.String())
			}
		})
	}
}
