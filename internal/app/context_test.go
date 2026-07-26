package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/testkit"
)

func TestNewRespectsCWD(t *testing.T) {
	home := testkit.TempHome(t)
	proj := filepath.Join(home, "proj")
	testkit.CopyFixture(t, "projects/empty-package-json", proj)

	ac, err := app.New(context.Background(), app.Options{CWD: proj})
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(proj)
	if ac.CWD != abs {
		t.Fatalf("CWD=%q want %q", ac.CWD, abs)
	}
	if ac.Config == nil {
		t.Fatal("nil config")
	}
}

func TestFromContextRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ac, err := app.New(context.Background(), app.Options{CWD: dir, Version: "1.2.3"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := app.WithContext(context.Background(), ac)
	got := app.FromContext(ctx)
	if got == nil || got.Version != "1.2.3" {
		t.Fatalf("%+v", got)
	}
}
