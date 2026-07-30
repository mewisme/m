package runner_test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestRunnerImportBoundaries(t *testing.T) {
	t.Helper()
	cases := []struct {
		pkg       string
		forbidden []string
	}{
		{"github.com/mewisme/mew/internal/runner", []string{"github.com/mewisme/mew/internal/runner/envexec"}},
		{"github.com/mewisme/mew/internal/runner/envexec", []string{"github.com/mewisme/mew/internal/app", "github.com/mewisme/mew/internal/cli"}},
		{"github.com/mewisme/mew/internal/conformance", []string{"github.com/mewisme/mew/internal/resolver", "github.com/mewisme/mew/internal/linker"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.pkg, func(t *testing.T) {
			cmd := exec.Command("go", "list", "-json", tc.pkg)
			out, err := cmd.Output()
			if err != nil {
				t.Fatal(err)
			}
			var pkg struct {
				ImportPath string
				Imports    []string
			}
			if err := json.Unmarshal(out, &pkg); err != nil {
				t.Fatal(err)
			}
			for _, imp := range pkg.Imports {
				for _, bad := range tc.forbidden {
					if imp == bad || strings.HasPrefix(imp, bad+"/") {
						t.Errorf("%s must not import %s", tc.pkg, imp)
					}
				}
			}
		})
	}
}
