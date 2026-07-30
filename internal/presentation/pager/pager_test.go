package pager_test

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/presentation/pager"
)

func TestSplitCommandRejectsShellMeta(t *testing.T) {
	_, _, err := pager.SplitCommand("less; rm -rf /")
	if !apperr.IsUsage(err) {
		t.Fatalf("want usage, got %v", err)
	}
}

func TestSplitCommandQuotes(t *testing.T) {
	path, args, err := pager.SplitCommand(`less "-R" '-F'`)
	if err != nil {
		t.Fatal(err)
	}
	if path != "less" || len(args) != 2 || args[0] != "-R" || args[1] != "-F" {
		t.Fatalf("path=%q args=%v", path, args)
	}
}

func TestResolveNever(t *testing.T) {
	plan, err := pager.Resolve(pager.Input{Flag: "never", PAGER: "less", StdoutTTY: true, Human: true, LineCount: 100})
	if err != nil || plan.Use {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestResolveAutoMissingPager(t *testing.T) {
	plan, err := pager.Resolve(pager.Input{
		Flag: "auto", MEWPager: "definitely-not-a-pager-bin-xyz",
		StdoutTTY: true, Human: true, LineCount: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Use {
		t.Fatalf("auto should not use missing pager: %+v", plan)
	}
}

func TestResolveAlwaysMissingPager(t *testing.T) {
	_, err := pager.Resolve(pager.Input{
		Flag: "always", MEWPager: "definitely-not-a-pager-bin-xyz",
		StdoutTTY: true, Human: true, LineCount: 100,
	})
	if apperr.CodeOf(err) != apperr.IO {
		t.Fatalf("want IO, got %v", err)
	}
}

func TestResolveWindowsNoDefaultLess(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only default policy")
	}
	plan, err := pager.Resolve(pager.Input{
		Flag: "auto", StdoutTTY: true, Human: true, LineCount: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Use {
		t.Fatalf("windows auto must not assume less: %+v", plan)
	}
}

func TestWritePagedDirect(t *testing.T) {
	var buf strings.Builder
	err := pager.WritePaged(context.Background(), &buf, "hello\n", pager.Plan{Mode: pager.ModeAuto})
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "hello\n" {
		t.Fatalf("got %q", buf.String())
	}
}

func TestLineCount(t *testing.T) {
	if pager.LineCount("a\nb\nc\n") != 3 {
		t.Fatal(pager.LineCount("a\nb\nc\n"))
	}
}
