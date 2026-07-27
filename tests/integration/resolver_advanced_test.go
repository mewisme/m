package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/policy"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/testkit"
)

func advancedEngine(t testing.TB, registryRel string) (*resolver.Engine, string) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")

	reg := testkit.LoadRegistry(t, registryRel)
	srv := reg.Start(t)
	cacheDir := filepath.Join(t.TempDir(), "registry-cache")
	client := registry.NewClient(registry.Options{
		BaseURL:    srv.URL,
		CacheDir:   cacheDir,
		HTTPClient: srv.Client(),
	})
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:        t.TempDir(),
		GlobalPath: filepath.Join(t.TempDir(), "missing-global.jsonc"),
		Env:        []string{},
		CLI:        map[string]any{"registry": srv.URL},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resolver.NewEngine(client, eff, project.IdentityMew), srv.URL
}

func copyFixturePackageJSON(t testing.TB, fixtureRel, pkgName string) string {
	t.Helper()
	dir := t.TempDir()
	src := testkit.FixtureDir(t, fixtureRel)
	data, err := os.ReadFile(filepath.Join(src, pkgName))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAdvancedResolverPeerContextFixture(t *testing.T) {
	eng, _ := advancedEngine(t, "resolver/peers/react-ecosystem/registry")
	root := copyFixturePackageJSON(t, "resolver/peers/react-ecosystem", "package-with-peers.json")

	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var reactKey string
	for _, p := range res.Graph.Packages {
		if p.ID.Name != "react" {
			continue
		}
		if len(p.ID.PeerProviderContext) == 0 {
			t.Fatalf("react missing peer providers: %#v", p.ID)
		}
		reactKey = p.ID.Key()
	}
	if reactKey != "react@18.2.0#react-dom@18.2.0" {
		t.Fatalf("react key=%q", reactKey)
	}
}

func TestAdvancedResolverStrictPeerFixture(t *testing.T) {
	eng, _ := advancedEngine(t, "resolver/peers/react-ecosystem/registry")
	root := copyFixturePackageJSON(t, "resolver/peers/react-ecosystem", "package.json")

	_, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: &policy.Policy{StrictPeerDependencies: true},
	})
	if err == nil {
		t.Fatal("expected strict peer error")
	}
	if apperr.CodeOf(err) != apperr.Resolve {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "react-dom") {
		t.Fatalf("want react-dom in error: %v", err)
	}
}

func TestAdvancedResolverAutoInstallPeersFixture(t *testing.T) {
	eng, _ := advancedEngine(t, "resolver/peers/react-ecosystem/registry")
	root := copyFixturePackageJSON(t, "resolver/peers/react-ecosystem", "package.json")

	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: &policy.Policy{AutoInstallPeers: true, StrictPeerDependencies: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	foundDOM := false
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "react-dom" {
			foundDOM = true
		}
	}
	if !foundDOM {
		t.Fatal("react-dom should be auto-installed")
	}
}

func TestAdvancedResolverOptionalPlatformFixture(t *testing.T) {
	eng, _ := advancedEngine(t, "resolver/optional-platform/registry")
	root := copyFixturePackageJSON(t, "resolver/optional-platform", "package.json")

	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if resolver.CurrentTarget().OS == "darwin" {
		found := false
		for _, p := range res.Graph.Packages {
			if p.ID.Name == "opt-darwin" {
				found = true
			}
		}
		if !found {
			t.Fatal("darwin-only optional should be installed on darwin")
		}
		return
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "opt-darwin" {
			t.Fatalf("darwin-only optional should be skipped on this host: %#v", p)
		}
	}
	skipped := false
	for _, d := range res.Decisions {
		if d.Package == "opt-darwin" && d.Reason == "platform-skipped" {
			skipped = true
		}
	}
	if !skipped {
		t.Fatal("expected platform-skipped decision for opt-darwin")
	}
}

func TestAdvancedResolverOptionalTransitiveFixture(t *testing.T) {
	eng, _ := advancedEngine(t, "resolver/optional-platform/registry")
	root := copyFixturePackageJSON(t, "resolver/optional-platform", "package-transitive.json")

	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "missing-opt" {
			t.Fatal("optional transitive failure should not add package")
		}
	}
	found := false
	for _, d := range res.Decisions {
		if d.Package == "missing-opt" && d.Reason == "optional-failed" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected optional-failed decision for missing-opt")
	}
}

func TestAdvancedResolverNestedOverrideFixture(t *testing.T) {
	eng, _ := advancedEngine(t, "registry/v1")
	root := copyFixturePackageJSON(t, "resolver/overrides-nested", "package.json")

	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "pkg-b" && p.ID.Version != "1.0.0" {
			t.Fatalf("nested override ignored: pkg-b@%s", p.ID.Version)
		}
	}
}

func TestAdvancedResolverWorkspaceProtocolFixture(t *testing.T) {
	eng, _ := advancedEngine(t, "registry/v1")
	projDir := t.TempDir()
	testkit.CopyFixture(t, "projects/workspace-protocol", projDir)

	res, err := eng.Resolve(context.Background(), projDir, resolver.ResolveOptions{
		Policy: &policy.Policy{StrictPeerDependencies: false},
	})
	if err != nil {
		t.Fatal(err)
	}

	foundLib := false
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "lib" && p.ID.Version == "2.4.0" {
			foundLib = true
			if p.Integrity != "" || p.TarballURL != "" {
				t.Fatalf("workspace package should have empty dist fields: %#v", p)
			}
		}
	}
	if !foundLib {
		t.Fatalf("missing workspace lib: %#v", res.Graph.Packages)
	}

	importers := map[string]bool{}
	for _, im := range res.Graph.Importers {
		importers[string(im.ID)] = true
	}
	if !importers["."] || !importers["packages/app"] {
		t.Fatalf("importers=%v", importers)
	}

	locals, err := resolver.DecodeLocalSources(res.Extensions)
	if err != nil {
		t.Fatal(err)
	}
	if src, ok := locals["lib@2.4.0"]; !ok || src.Protocol != "workspace" || src.Path != "packages/lib" {
		t.Fatalf("local extension=%#v", locals)
	}

	hasCaret := false
	for _, e := range res.Graph.Edges {
		if e.From == "packages/app" && strings.HasPrefix(e.To, "lib@") && e.Range == "workspace:^" {
			hasCaret = true
		}
	}
	if !hasCaret {
		t.Fatalf("missing workspace:^ edge: %#v", res.Graph.Edges)
	}
}

func TestAdvancedResolverIncrementalUpdateFixture(t *testing.T) {
	eng, _ := advancedEngine(t, "registry/v1")
	root := writeResolverProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "^4.17.0" }
}`)

	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "sha256-a", "http://example/pkg-a.tgz").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.0.0"}, "sha256-b", "http://example/pkg-b.tgz").
		Package(graph.PackageID{Name: "pkg-c", Version: "1.0.0"}, "sha256-c", "http://example/pkg-c.tgz").
		Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "sha256-l", "http://example/lodash.tgz").
		EdgeEx(string(graph.RootImporter), "pkg-a@1.0.0", graph.DepProd, "^1.0.0", false).
		EdgeEx(string(graph.RootImporter), "lodash@4.17.21", graph.DepProd, "^4.17.0", false).
		EdgeEx("pkg-a@1.0.0", "pkg-b@1.0.0", graph.DepProd, "^1.0.0", false).
		EdgeEx("pkg-b@1.0.0", "pkg-c@1.0.0", graph.DepProd, "^1.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Prior:             prior,
		Hints:             prior,
		UpdateTargets:     []string{"pkg-a"},
		IncrementalUpdate: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	versions := map[string]string{}
	for _, p := range res.Graph.Packages {
		versions[p.ID.Name] = p.ID.Version
	}
	if versions["pkg-b"] != "1.2.0" {
		t.Fatalf("pkg-b=%q want 1.2.0", versions["pkg-b"])
	}
	if versions["lodash"] != "4.17.21" {
		t.Fatalf("lodash=%q want pinned 4.17.21", versions["lodash"])
	}
}

func writeResolverProject(t testing.TB, pkgJSON string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
