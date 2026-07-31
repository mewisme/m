package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/cli"
)

func TestOutputPlainNoANSIOnPipe(t *testing.T) {
	root := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	code := cli.ExecuteWithArgv(root, nil, []string{"version", "--output=plain"})
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, errb.String())
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("stdout contains ANSI: %q", out.String())
	}
}
