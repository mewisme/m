package app

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/testkit"
)

func TestSelectExecImporterRejectsRecursive(t *testing.T) {
	ac := &Context{CWD: t.TempDir()}
	_, err := SelectExecImporter(context.Background(), ac, ExecImporterOptions{Recursive: true})
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("err=%v", err)
	}
}

func TestExecImporterSingleFilter(t *testing.T) {
	testkit.CleanEnv(t)
	testkit.EnableWorkspaces(t)
	root := testkit.SetupExecWorkspaceFixture(t, "tool")
	ac := &Context{CWD: root}
	_, err := SelectExecImporter(context.Background(), ac, ExecImporterOptions{Filters: []string{"missing-package"}})
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("err=%v", err)
	}
}
