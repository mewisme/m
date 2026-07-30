package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
)

func TestRunMissingScript(t *testing.T) {
	modRoot := testModuleRoot(t)
	fixture := filepath.Join(modRoot, "fixtures", "projects", "empty-package-json")
	ac, err := app.New(context.Background(), app.Options{
		CWD:      fixture,
		Reporter: diagnostics.NewReporter(diagnostics.Options{Format: "silent", Color: diagnostics.ColorNever}),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.Run(context.Background(), ac, app.RunOptions{Selector: "missing"})
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestRunIfPresent(t *testing.T) {
	modRoot := testModuleRoot(t)
	fixture := filepath.Join(modRoot, "fixtures", "projects", "empty-package-json")
	ac, err := app.New(context.Background(), app.Options{
		CWD:      fixture,
		Reporter: diagnostics.NewReporter(diagnostics.Options{Format: "silent", Color: diagnostics.ColorNever}),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.Run(context.Background(), ac, app.RunOptions{Selector: "missing", IfPresent: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testModuleRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
