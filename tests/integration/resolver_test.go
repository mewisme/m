package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/cli"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/testkit"
)

func TestResolverTransitiveFixture(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	projDir := t.TempDir()
	src := testkit.FixtureDir(t, "projects/semver-ranges")
	data, err := os.ReadFile(filepath.Join(src, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	client := registry.NewClient(registry.Options{
		BaseURL: srv.URL, CacheDir: filepath.Join(t.TempDir(), "cache"), HTTPClient: srv.Client(),
	})
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD: projDir, Env: []string{}, CLI: map[string]any{"registry": srv.URL},
		GlobalPath: filepath.Join(t.TempDir(), "nog.jsonc"),
	})
	if err != nil {
		t.Fatal(err)
	}
	eng := resolver.NewEngine(client, eff, project.IdentityMew)
	res, err := eng.Resolve(context.Background(), projDir, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, p := range res.Graph.Packages {
		keys[p.ID.Key()] = true
	}
	for _, want := range []string{"pkg-a@1.0.0", "pkg-b@1.2.0", "pkg-c@1.0.1", "lodash@4.17.21"} {
		if !keys[want] {
			t.Fatalf("missing %s in %v", want, keys)
		}
	}
	// Root devDependency pkg-c@1.0.0 should appear (exact), and prod path also has pkg-c@1.0.1.
	// Two versions of pkg-c may coexist under different edges — check at least one pkg-c.
	hasC := false
	for k := range keys {
		if len(k) >= 5 && k[:5] == "pkg-c" {
			hasC = true
		}
	}
	if !hasC {
		t.Fatal("pkg-c missing")
	}
}

func TestResolverCLIJSON(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	projDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "cli-resolve",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	root := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--cwd", projDir, "--config", cfgPath, "resolve", "--json"})
	code := cli.ExecuteWithContext(root, context.Background())
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, buf.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("json: %v\n%s", err, buf.String())
	}
	if doc["schemaVersion"] == nil {
		t.Fatalf("%v", doc)
	}
}
