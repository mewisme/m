package isolated_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/linker/isolated"
	"github.com/mewisme/mew/internal/testkit"
)

type linkerFixture struct {
	Packages []struct {
		Name          string `json:"name"`
		Version       string `json:"version"`
		PeerProviders []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Key     string `json:"key"`
		} `json:"peerProviders,omitempty"`
	} `json:"packages"`
	Edges []struct {
		From string `json:"from"`
		Name string `json:"name,omitempty"`
		To   string `json:"to"`
		Kind string `json:"kind"`
	} `json:"edges"`
	Expect                 map[string]string `json:"expect"`
	ExpectDistinctStoreIDs []string          `json:"expectDistinctStoreIDs,omitempty"`
}

func TestIsolatedLinkerFixtures(t *testing.T) {
	fixtures := []string{"alias-target", "dual-peer-context"}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			runIsolatedFixture(t, testkit.FixtureDir(t, filepath.Join("linker", "isolated", name, "graph.json")))
		})
	}
}

func runIsolatedFixture(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fx linkerFixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	g, err := graphFromFixture(&fx)
	if err != nil {
		t.Fatal(err)
	}
	stage := t.TempDir()
	nm := filepath.Join(stage, "node_modules")
	got, err := isolated.PackageContentDirs(g, nm)
	if err != nil {
		t.Fatal(err)
	}
	for key, wantRel := range fx.Expect {
		want := filepath.Join(stage, filepath.FromSlash(wantRel))
		dest, ok := got[key]
		if !ok {
			t.Fatalf("%s: missing placement", key)
		}
		if filepath.Clean(dest) != filepath.Clean(want) {
			t.Fatalf("%s: got %q want %q", key, dest, want)
		}
	}
	if len(fx.ExpectDistinctStoreIDs) > 1 {
		ids := map[string]string{}
		for _, key := range fx.ExpectDistinctStoreIDs {
			id := isolated.StoreIDFromKey(key)
			if prev, ok := ids[id]; ok {
				t.Fatalf("store id collision: %s and %s -> %s", prev, key, id)
			}
			ids[id] = key
		}
	}
}

func graphFromFixture(fx *linkerFixture) (*graph.Graph, error) {
	b := graph.NewBuilder().Importer(graph.RootImporter, "root")
	for _, p := range fx.Packages {
		id := graph.PackageID{Name: p.Name, Version: p.Version}
		for _, peer := range p.PeerProviders {
			id.PeerProviderContext = append(id.PeerProviderContext, graph.PeerProvider{
				Name: peer.Name, Version: peer.Version, Key: peer.Key,
			})
		}
		id.Normalize()
		b = b.Package(id, "", "")
	}
	for _, e := range fx.Edges {
		name := e.Name
		if name == "" {
			name = graph.TargetNameFromKey(e.To)
		}
		b = b.EdgeEx(e.From, name, e.To, graph.DepKind(e.Kind), "", false)
	}
	return b.Build()
}
