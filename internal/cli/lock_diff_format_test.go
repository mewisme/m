package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/cli"
	"github.com/mewisme/mew/internal/lockfile"
)

func TestFormatLockDiffHuman(t *testing.T) {
	var buf bytes.Buffer
	diff := &lockfile.GraphDiff{
		PackagesAdded:   []string{"a@2.0.0"},
		PackagesRemoved: []string{"a@1.0.0"},
		Specifiers: []lockfile.ImporterSpecifierDiff{
			{Importer: ".", Name: "a", Kind: "prod", Before: "^1.0.0", After: "^2.0.0"},
		},
	}
	if err := cli.FormatLockDiffHuman(&buf, diff); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"+ a  2.0.0", "- a  1.0.0", "~ . a prod: ^1.0.0 -> ^2.0.0"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestFormatLockDiffHumanEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := cli.FormatLockDiffHuman(&buf, &lockfile.GraphDiff{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "no changes") {
		t.Fatalf("out=%q", buf.String())
	}
}
