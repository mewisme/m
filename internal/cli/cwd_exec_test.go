package cli

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/testkit"
)

func TestExecGlobalCWDViaExecuteWithContext(t *testing.T) {
	testkit.CleanEnv(t)
	proj := testkit.SetupExecFixture(t, "tool")
	absProj, err := filepath.Abs(proj)
	if err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	var gotCWD string
	root.AddCommand(&cobra.Command{
		Use: "print-cwd",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac != nil {
				gotCWD = ac.CWD
			}
			return nil
		},
	})

	out := new(bytes.Buffer)
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"--cwd", absProj, "print-cwd"})
	code := ExecuteWithContext(root, context.Background())
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if gotCWD != absProj {
		t.Fatalf("gotCWD=%q want %q", gotCWD, absProj)
	}
}

func TestExecSubcommandGlobalCWD(t *testing.T) {
	testkit.CleanEnv(t)
	proj := testkit.SetupExecFixture(t, "tool")
	absProj, err := filepath.Abs(proj)
	if err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	out := new(bytes.Buffer)
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"--cwd", absProj, "exec", "tool"})
	code := ExecuteWithContext(root, context.Background())
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out.String())
	}
}

func TestCWDViaRootExecute(t *testing.T) {
	testkit.CleanEnv(t)
	proj := testkit.SetupExecFixture(t, "tool")
	absProj, err := filepath.Abs(proj)
	if err != nil {
		t.Fatal(err)
	}

	root := NewMRoot(testBuildInfo())
	var gotCWD string
	root.AddCommand(&cobra.Command{
		Use: "print-cwd",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac != nil {
				gotCWD = ac.CWD
			}
			return nil
		},
	})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"--cwd", absProj, "print-cwd"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if gotCWD != absProj {
		t.Fatalf("gotCWD=%q want %q", gotCWD, absProj)
	}
}
