package pnpm

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

func refencodeFixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
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

func TestEncodeDependencyRefGolden(t *testing.T) {
	cases := []struct {
		depName, targetKey, want string
	}{
		{"b", "b@2.0.0", "2.0.0"},
		{"acorn-jsx", "acorn-jsx@5.3.2#acorn@8.18.0", "5.3.2(acorn@8.18.0)"},
		{"local", "link:../pkg", "link:../pkg"},
		{"alias", "npm:lodash@4.17.21", "npm:lodash@4.17.21"},
	}
	for _, tc := range cases {
		got, err := EncodeDependencyRef(tc.depName, tc.targetKey)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.depName, tc.targetKey, err)
		}
		if got != tc.want {
			t.Fatalf("%s %s: got %q want %q", tc.depName, tc.targetKey, got, tc.want)
		}
	}
}

func TestDependencyRefRoundTripFixtures(t *testing.T) {
	root := refencodeFixtureRoot(t)
	families := []struct {
		major  int
		family string
	}{
		{9, "basic"},
		{9, "peer-context"},
		{10, "basic"},
		{10, "peer-context"},
		{11, "basic"},
		{11, "peer-context"},
	}
	for _, tc := range families {
		t.Run(fmt.Sprintf("pnpm-%d-%s", tc.major, tc.family), func(t *testing.T) {
			path := filepath.Join(root, "fixtures", "locks", "generated",
				fmt.Sprintf("pnpm-%d", tc.major), tc.family, "pnpm-lock.yaml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := Decode(data)
			if err != nil {
				t.Fatal(err)
			}
			g, err := ToGraph(doc)
			if err != nil {
				t.Fatal(err)
			}
			out, err := FromGraph(g, doc, lockfile.Detection{
				Format: FormatV9, ProducerMajor: tc.major, ExplicitMajor: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			assertEncodedImporterRefs(t, doc, out)
			assertEncodedSnapshotRefs(t, g, out)
		})
	}
}

func assertEncodedImporterRefs(t *testing.T, prior, out *Document) {
	t.Helper()
	for id, sec := range prior.Importers {
		outSec, ok := out.Importers[id]
		if !ok {
			t.Fatalf("missing importer %q", id)
		}
		for name, dep := range sec.Dependencies {
			got := outSec.Dependencies[name]
			if got.Version != dep.Version {
				t.Fatalf("importer %q dep %q: got version %q want %q", id, name, got.Version, dep.Version)
			}
		}
		for name, dep := range sec.DevDependencies {
			got := outSec.DevDependencies[name]
			if got.Version != dep.Version {
				t.Fatalf("importer %q devDep %q: got version %q want %q", id, name, got.Version, dep.Version)
			}
		}
		for name, dep := range sec.OptionalDependencies {
			got := outSec.OptionalDependencies[name]
			if got.Version != dep.Version {
				t.Fatalf("importer %q optDep %q: got version %q want %q", id, name, got.Version, dep.Version)
			}
		}
	}
}

func assertEncodedSnapshotRefs(t *testing.T, g *graph.Graph, out *Document) {
	t.Helper()
	for _, p := range g.Packages {
		graphKey := p.ID.Key()
		instanceKey, err := graphKeyToInstanceKey(graphKey)
		if err != nil {
			t.Fatal(err)
		}
		snap, ok := out.Snapshots[instanceKey]
		if !ok {
			t.Fatalf("missing snapshot %q", instanceKey)
		}
		for _, field := range []string{"dependencies", "optionalDependencies", "peerDependencies"} {
			raw, ok := snap[field]
			if !ok {
				continue
			}
			m, ok := raw.(map[string]string)
			if !ok {
				t.Fatalf("snapshot %q field %q not a string map", instanceKey, field)
			}
			for depName, ref := range m {
				if strings.Contains(ref, "@") && !isProtocolRef(ref) {
					if strings.HasPrefix(ref, depName+"@") {
						t.Fatalf("snapshot %q dep %q double-prefixed ref %q", instanceKey, depName, ref)
					}
				}
			}
		}
	}
}
