package resolver_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/policy"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/testkit"
)

func testEngine(t testing.TB) (*resolver.Engine, string) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("no_proxy", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	cacheDir := filepath.Join(t.TempDir(), "registry-cache")
	client := registry.NewClient(registry.Options{
		BaseURL:    srv.URL,
		CacheDir:   cacheDir,
		HTTPClient: srv.Client(),
	})
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:         t.TempDir(),
		GlobalPath:  filepath.Join(t.TempDir(), "missing-global.jsonc"),
		ProjectPath: filepath.Join(t.TempDir(), "missing-project.jsonc"),
		Env:         []string{},
		CLI:         map[string]any{"registry": srv.URL},
	})
	if err != nil {
		t.Fatal(err)
	}
	eng := resolver.NewEngine(client, eff, project.IdentityMew)
	return eng, srv.URL
}

func priorFingerprints(eng *resolver.Engine) *resolver.PriorFingerprints {
	pol := resolver.PolicyFromEffective(eng.Effective)
	return &resolver.PriorFingerprints{
		ResolverPolicyFingerprint: resolver.PolicyFingerprint(pol),
		TargetPlatformFingerprint: resolver.TargetPlatformFingerprint(resolver.CurrentTarget()),
	}
}

func writeProject(t testing.TB, pkgJSON string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestResolveTransitiveABC(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, p := range res.Graph.Packages {
		keys[p.ID.Key()] = true
	}
	for _, want := range []string{"pkg-a@1.0.0", "pkg-b@1.2.0", "pkg-c@1.0.1"} {
		if !keys[want] {
			t.Fatalf("missing %s in %#v", want, keys)
		}
	}
}

func TestResolveCaretHighest(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-b": "^1.0.0" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "pkg-b" {
			found = true
			if p.ID.Version != "1.2.0" {
				t.Fatalf("got pkg-b@%s want 1.2.0", p.ID.Version)
			}
		}
	}
	if !found {
		t.Fatal("pkg-b missing")
	}
}

func TestResolveDeterministic(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "~4.17.0" }
}`)
	r1, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	b1, err := graph.EncodeJSON(r1.Graph)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := graph.EncodeJSON(r2.Graph)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatalf("non-deterministic\n%s\n%s", b1, b2)
	}
	goldenPath := filepath.Join("..", "..", "testdata", "resolver", "golden", "graphs", "transitive-a-b-c.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		norm := stripTarballHosts(b1)
		if err := os.WriteFile(goldenPath, norm, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (set UPDATE_GOLDEN=1 to create): %v", err)
	}
	gotNorm := stripTarballHosts(b1)
	wantNorm := stripTarballHosts(want)
	if !bytes.Equal(gotNorm, wantNorm) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", gotNorm, wantNorm)
	}
}

func stripTarballHosts(b []byte) []byte {
	s := string(b)
	out := strings.Builder{}
	for {
		i := strings.Index(s, "http://127.0.0.1:")
		if i < 0 {
			out.WriteString(s)
			break
		}
		out.WriteString(s[:i])
		out.WriteString("http://registry.test")
		rest := s[i+len("http://127.0.0.1:"):]
		j := strings.IndexByte(rest, '/')
		if j < 0 {
			out.WriteString(rest)
			break
		}
		s = rest[j:]
	}
	return []byte(out.String())
}

func TestResolveUnsatisfiable(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-b": "^9.0.0" }
}`)
	_, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Resolve {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "pkg-b") {
		t.Fatalf("want package name in error: %v", err)
	}
}

func TestResolveValidCycle(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "cycle-a": "1.0.0" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatalf("valid same-key cycle should resolve: %v", err)
	}
	foundA, foundB := false, false
	for _, p := range res.Graph.Packages {
		switch p.ID.Name {
		case "cycle-a":
			foundA = true
		case "cycle-b":
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("expected cycle-a and cycle-b in graph: %#v", res.Graph.Packages)
	}
}

func TestResolveHints(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-b": "^1.0.0" }
}`)
	hints, err := graph.NewBuilder().
		Package(graph.PackageID{Name: "pkg-b", Version: "1.0.0"}, "sha256-hint", "http://example/pkg-b.tgz").
		Package(graph.PackageID{Name: "pkg-c", Version: "1.0.0"}, "sha256-c", "http://example/pkg-c.tgz").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{Hints: hints})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "pkg-b" && p.ID.Version != "1.0.0" {
			t.Fatalf("hint ignored: got %s", p.ID.Version)
		}
	}
	for _, d := range res.Decisions {
		if d.Package == "pkg-b" && d.Reason != "hint" {
			t.Fatalf("decision reason=%q", d.Reason)
		}
	}
}

func TestResolveDevDepsRootOnly(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "1.0.0" },
  "devDependencies": { "lodash": "4.17.21" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var hasLodash, hasDevEdge bool
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "lodash" {
			hasLodash = true
		}
	}
	for _, e := range res.Graph.Edges {
		if e.Kind == graph.DepDev && strings.HasPrefix(e.To, "lodash@") {
			hasDevEdge = true
		}
	}
	if !hasLodash || !hasDevEdge {
		t.Fatalf("dev dep missing: packages=%v edges=%v", hasLodash, hasDevEdge)
	}
}

func TestResolveOmitRootDev(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "1.0.0" },
  "devDependencies": { "lodash": "4.17.21" }
}`)
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{OmitRootDev: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "lodash" {
			t.Fatalf("lodash should be omitted with OmitRootDev: %#v", res.Graph.Packages)
		}
	}
	for _, e := range res.Graph.Edges {
		if e.Kind == graph.DepDev {
			t.Fatalf("unexpected dev edge: %#v", e)
		}
	}
	keys := map[string]bool{}
	for _, p := range res.Graph.Packages {
		keys[p.ID.Key()] = true
	}
	for _, want := range []string{"pkg-a@1.0.0", "pkg-b@1.2.0", "pkg-c@1.0.1"} {
		if !keys[want] {
			t.Fatalf("missing prod transitive %s in %#v", want, keys)
		}
	}
}

func TestResolveRejectDeprecated(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-c": "1.1.0-beta.1" }
}`)
	pol := &policy.Policy{RejectDeprecated: true}
	_, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{Policy: pol})
	if err == nil {
		t.Fatal("expected reject")
	}
	if apperr.CodeOf(err) != apperr.Resolve {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}

func TestResolveMinimumReleaseAge(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-c": "1.1.0-beta.1" }
}`)
	pol := &policy.Policy{MinimumReleaseAge: 24 * time.Hour}
	_, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{Policy: pol})
	if err == nil {
		t.Fatal("expected age reject for 2099 publish")
	}
}

func BenchmarkResolveTransitive(b *testing.B) {
	eng, _ := testEngine(b)
	root := writeProject(b, `{
  "name": "root", "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0" }
}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{}); err != nil {
			b.Fatal(err)
		}
	}
}
