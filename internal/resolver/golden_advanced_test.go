package resolver_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/testkit"
)

func TestGoldenPeersReactEcosystem(t *testing.T) {
	eng, _ := engineWithPackuments(t, reactPackuments())
	root := writeProject(t, readFixture(t, "resolver/peers/react-ecosystem/package-with-peers.json"))
	assertGoldenGraph(t, "peers-react-ecosystem.json", eng, root, resolver.ResolveOptions{})
}

func TestGoldenOverridesNested(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, readFixture(t, "resolver/overrides-nested/package.json"))
	assertGoldenGraph(t, "overrides-nested.json", eng, root, resolver.ResolveOptions{})
}

func TestGoldenWorkspaceProtocol(t *testing.T) {
	eng, _ := testEngine(t)
	projDir := t.TempDir()
	testkit.CopyFixture(t, "projects/workspace-protocol", projDir)
	assertGoldenGraph(t, "workspace-protocol.json", eng, projDir, resolver.ResolveOptions{
		Policy: testStrictPeersOff(),
	})
}

func readFixture(t testing.TB, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(testkit.FixtureDir(t, rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertGoldenGraph(t testing.TB, name string, eng *resolver.Engine, root string, opts resolver.ResolveOptions) {
	t.Helper()
	res, err := eng.Resolve(context.Background(), root, opts)
	if err != nil {
		t.Fatal(err)
	}
	got, err := graph.EncodeJSON(res.Graph)
	if err != nil {
		t.Fatal(err)
	}
	goldenPath := filepath.Join("..", "..", "testdata", "resolver", "golden", "graphs", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		norm := stripTarballHosts(got)
		if err := os.WriteFile(goldenPath, norm, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s (set UPDATE_GOLDEN=1 to create): %v", name, err)
	}
	gotNorm := stripTarballHosts(got)
	wantNorm := stripTarballHosts(want)
	if !bytes.Equal(gotNorm, wantNorm) {
		t.Fatalf("golden %s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, gotNorm, wantNorm)
	}
}
