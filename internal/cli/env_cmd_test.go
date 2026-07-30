package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func TestEnvInspectProjectGlobalCWD(t *testing.T) {
	testkit.CleanEnv(t)
	proj := testkit.SetupExecFixture(t, "echo-bin")
	absProj, err := filepath.Abs(proj)
	if err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	out := new(bytes.Buffer)
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"--cwd", absProj, "env", "inspect", "project", "echo-bin"})
	code := ExecuteWithContext(root, context.Background())
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out.String())
	}
	if !strings.Contains(out.String(), `"source": "project"`) {
		t.Fatalf("out=%s", out.String())
	}
}
