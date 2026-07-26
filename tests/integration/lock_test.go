package integration_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/cli"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/lockfile/mlock"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/testkit"
)

func TestLockResolveWriteRead(t *testing.T) {
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)

	projDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "lock-test",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD: projDir, Env: []string{}, CLI: map[string]any{"registry": srv.URL},
		GlobalPath: filepath.Join(t.TempDir(), "nog.jsonc"),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = project.Open(context.Background(), projDir)
	if err != nil {
		t.Fatal(err)
	}
	eng := resolver.NewEngine(registry.NewClient(registry.Options{
		BaseURL: srv.URL, CacheDir: filepath.Join(t.TempDir(), "cache"), HTTPClient: srv.Client(),
	}), eff, project.IdentityMew)
	res, err := eng.Resolve(context.Background(), projDir, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}

	ac := &app.Context{CWD: projDir, Config: eff, Ctx: context.Background()}
	if err := app.WriteLock(context.Background(), ac, res); err != nil {
		t.Fatal(err)
	}

	g1, err := app.ReadLockGraph(context.Background(), ac)
	if err != nil {
		t.Fatal(err)
	}
	hints, err := eng.Resolve(context.Background(), projDir, resolver.ResolveOptions{Hints: g1})
	if err != nil {
		t.Fatal(err)
	}
	keys1 := packageKeys(g1)
	keys2 := packageKeys(hints.Graph)
	if len(keys1) != len(keys2) {
		t.Fatalf("key count %d vs %d", len(keys1), len(keys2))
	}
	for k := range keys1 {
		if !keys2[k] {
			t.Fatalf("missing %s after hint resolve", k)
		}
	}

	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	cliRoot.SetOut(buf)
	cliRoot.SetErr(buf)
	cliRoot.SetArgs([]string{"--cwd", projDir, "--config", cfgPath, "lock", "validate", "--frozen", "--json"})
	if code := cli.ExecuteWithContext(cliRoot, context.Background()); code != 0 {
		t.Fatalf("validate exit=%d out=%s", code, buf.String())
	}
}

func packageKeys(g *graph.Graph) map[string]bool {
	out := make(map[string]bool, len(g.Packages))
	for _, p := range g.Packages {
		out[p.ID.Key()] = true
	}
	return out
}

func TestMlockGreenfieldFixture(t *testing.T) {
	root := testkit.ModuleRoot(t)
	fixture := filepath.Join(root, "fixtures", "projects", "mlock-greenfield")
	data, err := os.ReadFile(filepath.Join(fixture, "m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := mlock.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if doc.LockfileVersion != mlock.LockfileVersion {
		t.Fatalf("version=%d", doc.LockfileVersion)
	}
	if doc.Checksum == "" {
		t.Fatal("missing checksum")
	}
}
