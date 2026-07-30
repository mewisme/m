package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/cli"
	"github.com/mewisme/mew/internal/presentation"
)

func TestCLIUXOutputModes(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		env   map[string]string
		check func(t *testing.T, code int, stdout, stderr string)
	}{
		{
			name: "plain-version-no-csi",
			args: []string{"version", "--output=plain"},
			check: func(t *testing.T, code int, stdout, stderr string) {
				t.Helper()
				if code != 0 {
					t.Fatalf("exit=%d stderr=%q", code, stderr)
				}
				assertNoCSI(t, stdout, stderr)
			},
		},
		{
			name: "accessible-version-no-csi",
			args: []string{"version", "--accessible"},
			check: func(t *testing.T, code int, stdout, stderr string) {
				t.Helper()
				if code != 0 {
					t.Fatalf("exit=%d stderr=%q", code, stderr)
				}
				assertNoCSI(t, stdout, stderr)
			},
		},
		{
			name: "ci-env-plain-no-csi",
			args: []string{"version"},
			env:  map[string]string{"CI": "1"},
			check: func(t *testing.T, code int, stdout, stderr string) {
				t.Helper()
				if code != 0 {
					t.Fatalf("exit=%d stderr=%q", code, stderr)
				}
				assertNoCSI(t, stdout, stderr)
			},
		},
		{
			name: "legacy-presentation-no-csi",
			args: []string{"version", "--presentation-legacy"},
			check: func(t *testing.T, code int, stdout, stderr string) {
				t.Helper()
				if code != 0 {
					t.Fatalf("exit=%d stderr=%q", code, stderr)
				}
				assertNoCSI(t, stdout, stderr)
			},
		},
		{
			name: "json-version-parseable",
			args: []string{"version", "--json"},
			check: func(t *testing.T, code int, stdout, stderr string) {
				t.Helper()
				if code != 0 {
					t.Fatalf("exit=%d stderr=%q", code, stderr)
				}
				assertNoCSI(t, stdout, stderr)
				var doc map[string]any
				if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &doc); err != nil {
					t.Fatalf("stdout not JSON: %v %q", err, stdout)
				}
				if _, ok := doc["binary"]; !ok {
					t.Fatalf("missing binary field: %q", stdout)
				}
			},
		},
		{
			name: "features-json-parseable",
			args: []string{"features", "--format", "json", "--module", "runner", "--status", "shipped"},
			check: func(t *testing.T, code int, stdout, stderr string) {
				t.Helper()
				if code != 0 {
					t.Fatalf("exit=%d stderr=%q", code, stderr)
				}
				assertNoCSI(t, stdout, stderr)
				var doc any
				if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &doc); err != nil {
					t.Fatalf("features JSON not parseable: %v %q", err, stdout)
				}
			},
		},
		{
			name: "help-plain-no-csi",
			args: []string{"--help", "--output=plain"},
			check: func(t *testing.T, code int, stdout, stderr string) {
				t.Helper()
				if code != 0 {
					t.Fatalf("exit=%d stderr=%q", code, stderr)
				}
				assertNoCSI(t, stdout, stderr)
			},
		},
		{
			name: "forced-rich-on-pipe-unsupported",
			args: []string{"version", "--output=rich"},
			check: func(t *testing.T, code int, stdout, stderr string) {
				t.Helper()
				if code == 0 {
					t.Fatal("expected rich on non-TTY to fail")
				}
				assertNoCSI(t, stdout, stderr)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			root := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
			var out, errb bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&errb)
			code := cli.ExecuteWithArgv(root, nil, tc.args)
			tc.check(t, code, out.String(), errb.String())
		})
	}
}

func assertNoCSI(t *testing.T, parts ...string) {
	t.Helper()
	for _, p := range parts {
		b := []byte(p)
		if presentation.ContainsCSI(b) {
			t.Fatalf("contains CSI: %q", p)
		}
		if presentation.ContainsCursorControl(b) {
			t.Fatalf("contains cursor control: %q", p)
		}
	}
}
