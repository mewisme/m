package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/binmeta"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/testkit"
)

func TestInstallPublishesBinMetadata(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	t.Cleanup(srv.Close)

	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := `{
  "name": "binmeta-install",
  "version": "1.0.0",
  "dependencies": { "pkg-cli": "1.0.0" }
}`
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	loadOpts := config.LoadOptions{
		CWD:         proj,
		ProjectRoot: proj,
		CLI:         map[string]any{"registry": srv.URL},
	}
	eff, err := config.Load(context.Background(), loadOpts)
	if err != nil {
		t.Fatal(err)
	}
	ac := &Context{CWD: proj, Config: eff, ConfigLoadSpec: config.LoadSpecFromOptions(loadOpts)}

	if _, err := Install(context.Background(), ac, InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	nm := filepath.Join(proj, "node_modules")
	if _, err := os.Stat(binmeta.Path(nm)); err != nil {
		t.Fatalf("bins metadata: %v", err)
	}
	bind, err := LoadGenerationBinding(proj)
	if err != nil {
		t.Fatal(err)
	}
	if bind.GenerationID == "" || bind.Fingerprint == "" {
		t.Fatalf("generation binding=%+v", bind)
	}
	doc, err := binmeta.Read(nm)
	if err != nil {
		t.Fatal(err)
	}
	if doc.GenerationID != bind.GenerationID || doc.Fingerprint != bind.Fingerprint {
		t.Fatalf("bind=%+v doc=%+v", bind, doc)
	}
	if len(doc.Records) == 0 {
		t.Fatal("expected bin records")
	}
}
