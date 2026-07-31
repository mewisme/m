package app

import (
	"fmt"
	"testing"
)

func TestDiffKeysNoChange(t *testing.T) {
	prior := map[string]string{"a@1.0.0": "1.0.0"}
	next := map[string]string{"a@1.0.0": "1.0.0"}
	res := diffKeys(prior, next)
	if res.Added != 0 || res.Removed != 0 || res.Changed != 0 {
		t.Fatalf("got added=%d removed=%d changed=%d want 0/0/0", res.Added, res.Removed, res.Changed)
	}
	if len(res.PackageChanges) != 0 {
		t.Fatalf("expected no changes, got %d", len(res.PackageChanges))
	}
}

func TestDiffKeysOneAddition(t *testing.T) {
	prior := map[string]string{}
	next := map[string]string{"zod@4.0.14": "4.0.14"}
	res := diffKeys(prior, next)
	if res.Added != 1 || res.Removed != 0 || res.Changed != 0 {
		t.Fatalf("got added=%d removed=%d changed=%d", res.Added, res.Removed, res.Changed)
	}
	if res.Packages != 1 {
		t.Fatalf("packages=%d", res.Packages)
	}
	if len(res.PackageChanges) != 1 {
		t.Fatalf("changes=%d", len(res.PackageChanges))
	}
	c := res.PackageChanges[0]
	if c.Kind != PackageChangeAdded {
		t.Fatalf("kind=%s", c.Kind)
	}
	if c.Name != "zod" || c.ToVersion != "4.0.14" {
		t.Fatalf("name=%s to=%s", c.Name, c.ToVersion)
	}
}

func TestDiffKeysOneRemoval(t *testing.T) {
	prior := map[string]string{"lodash@4.17.20": "4.17.20"}
	next := map[string]string{}
	res := diffKeys(prior, next)
	if res.Removed != 1 || res.Added != 0 || res.Changed != 0 {
		t.Fatalf("got added=%d removed=%d changed=%d", res.Added, res.Removed, res.Changed)
	}
	if len(res.PackageChanges) != 1 {
		t.Fatalf("changes=%d", len(res.PackageChanges))
	}
	c := res.PackageChanges[0]
	if c.Kind != PackageChangeRemoved {
		t.Fatalf("kind=%s", c.Kind)
	}
	if c.Name != "lodash" || c.FromVersion != "4.17.20" {
		t.Fatalf("name=%s from=%s", c.Name, c.FromVersion)
	}
}

func TestDiffKeysOneUpdate(t *testing.T) {
	prior := map[string]string{"react@19.1.0": "19.1.0"}
	next := map[string]string{"react@19.1.1": "19.1.1"}
	res := diffKeys(prior, next)
	if res.Changed != 1 || res.Added != 0 || res.Removed != 0 {
		t.Fatalf("got added=%d removed=%d changed=%d", res.Added, res.Removed, res.Changed)
	}
	if len(res.PackageChanges) != 1 {
		t.Fatalf("changes=%d", len(res.PackageChanges))
	}
	c := res.PackageChanges[0]
	if c.Kind != PackageChangeUpdated {
		t.Fatalf("kind=%s", c.Kind)
	}
	if c.Name != "react" || c.FromVersion != "19.1.0" || c.ToVersion != "19.1.1" {
		t.Fatalf("name=%s from=%s to=%s", c.Name, c.FromVersion, c.ToVersion)
	}
}

func TestDiffKeysScopedPackageAddition(t *testing.T) {
	prior := map[string]string{}
	next := map[string]string{"@types/react@18.3.0": "18.3.0"}
	res := diffKeys(prior, next)
	if res.Added != 1 {
		t.Fatalf("added=%d", res.Added)
	}
	c := res.PackageChanges[0]
	if c.Name != "@types/react" || c.ToVersion != "18.3.0" {
		t.Fatalf("name=%s to=%s", c.Name, c.ToVersion)
	}
}

func TestDiffKeysScopedPackageUpdate(t *testing.T) {
	prior := map[string]string{"@scope/pkg@1.0.0": "1.0.0"}
	next := map[string]string{"@scope/pkg@2.0.0": "2.0.0"}
	res := diffKeys(prior, next)
	if res.Changed != 1 {
		t.Fatalf("changed=%d", res.Changed)
	}
	c := res.PackageChanges[0]
	if c.Name != "@scope/pkg" || c.FromVersion != "1.0.0" || c.ToVersion != "2.0.0" {
		t.Fatalf("name=%s from=%s to=%s", c.Name, c.FromVersion, c.ToVersion)
	}
}

func TestDiffKeysMixedOperations(t *testing.T) {
	prior := map[string]string{
		"zod@4.0.13":      "4.0.13",
		"removed@1.0.0":   "1.0.0",
		"unchanged@1.0.0": "1.0.0",
	}
	next := map[string]string{
		"zod@4.0.14":      "4.0.14",
		"added@2.0.0":     "2.0.0",
		"unchanged@1.0.0": "1.0.0",
	}
	res := diffKeys(prior, next)
	if res.Added != 1 || res.Removed != 1 || res.Changed != 1 {
		t.Fatalf("added=%d removed=%d changed=%d", res.Added, res.Removed, res.Changed)
	}
	if len(res.PackageChanges) != 3 {
		t.Fatalf("changes=%d want 3", len(res.PackageChanges))
	}

	// Verify counts match changes.
	var added, removed, updated int
	for _, c := range res.PackageChanges {
		switch c.Kind {
		case PackageChangeAdded:
			added++
		case PackageChangeRemoved:
			removed++
		case PackageChangeUpdated:
			updated++
		}
	}
	if added != res.Added || removed != res.Removed || updated != res.Changed {
		t.Fatalf("count mismatch: delta added=%d removed=%d updated=%d, metric added=%d removed=%d changed=%d",
			added, removed, updated, res.Added, res.Removed, res.Changed)
	}
}

func TestDiffKeysMultipleVersionsOfSamePackage(t *testing.T) {
	prior := map[string]string{
		"react@18.0.0": "18.0.0",
		"react@19.1.0": "19.1.0",
	}
	next := map[string]string{
		"react@18.0.0": "18.0.0",
		"react@19.1.1": "19.1.1",
	}
	res := diffKeys(prior, next)
	if res.Changed != 1 || res.Added != 0 || res.Removed != 0 {
		t.Fatalf("added=%d removed=%d changed=%d", res.Added, res.Removed, res.Changed)
	}
}

func TestDiffKeysPeerContextKeysPaired(t *testing.T) {
	prior := map[string]string{
		"react@18.2.0(peer_a)": "18.2.0",
	}
	next := map[string]string{
		"react@18.3.0(peer_b)": "18.3.0",
	}
	res := diffKeys(prior, next)
	if res.Changed != 1 || res.Added != 0 || res.Removed != 0 {
		t.Fatalf("added=%d removed=%d changed=%d (peer-context keys should pair as update)", res.Added, res.Removed, res.Changed)
	}
	c := res.PackageChanges[0]
	if c.Kind != PackageChangeUpdated || c.Name != "react" {
		t.Fatalf("kind=%s name=%s", c.Kind, c.Name)
	}
}

func TestDiffKeysPairDeterministicOrder(t *testing.T) {
	prior := map[string]string{
		"react@18.0.0(a)": "18.0.0",
		"react@19.0.0(b)": "19.0.0",
	}
	next := map[string]string{
		"react@19.0.1(b)": "19.0.1",
		"react@18.0.1(a)": "18.0.1",
	}
	// Run multiple times to verify determinism.
	for i := 0; i < 20; i++ {
		res := diffKeys(prior, next)
		if res.Changed != 2 {
			t.Fatalf("run %d: changed=%d want 2", i, res.Changed)
		}
		if len(res.PackageChanges) != 2 {
			t.Fatalf("run %d: changes=%d", i, len(res.PackageChanges))
		}
		for _, c := range res.PackageChanges {
			if c.Kind != PackageChangeUpdated {
				t.Fatalf("run %d: kind=%s", i, c.Kind)
			}
		}
	}
}

func TestDiffKeysDeterministicOrderFromRandomInput(t *testing.T) {
	// Ensure deterministic output even with random map iteration.
	keys := []string{"z@2", "a@2", "y@1", "b@1"}
	for i := 0; i < 20; i++ {
		prior := map[string]string{}
		for _, k := range keys {
			prior[k] = k[len(k)-1:]
		}
		next := map[string]string{
			"added@1": "1",
			"added@2": "2",
		}
		res := diffKeys(prior, next)
		if res.Added != 2 || res.Removed != 4 {
			t.Fatalf("run %d: added=%d removed=%d", i, res.Added, res.Removed)
		}
		// Verify stable sort order.
		for i := 1; i < len(res.PackageChanges); i++ {
			prev := res.PackageChanges[i-1]
			cur := res.PackageChanges[i]
			if prev.Kind > cur.Kind {
				t.Fatalf("run %d: kind order broken", i)
			}
			if prev.Kind == cur.Kind && prev.Name > cur.Name {
				t.Fatalf("run %d: name order broken", i)
			}
		}
	}
}

func TestDiffKeysLocalAndWorkspaceDeps(t *testing.T) {
	prior := map[string]string{
		"workspace-pkg@file:../shared": "file:../shared",
	}
	next := map[string]string{
		"workspace-pkg@file:../shared": "file:../shared",
	}
	res := diffKeys(prior, next)
	if res.Added != 0 || res.Removed != 0 || res.Changed != 0 {
		t.Fatalf("unchanged local dep should produce no changes: added=%d removed=%d changed=%d", res.Added, res.Removed, res.Changed)
	}
}

func TestDiffKeysLargerThanMaxButCountsCorrect(t *testing.T) {
	n := 60
	prior := map[string]string{}
	next := map[string]string{}
	for i := 0; i < n; i++ {
		prior[fmt.Sprintf("old-pkg-%d@1.0.0", i)] = "1.0.0"
		next[fmt.Sprintf("new-pkg-%d@2.0.0", i)] = "2.0.0"
	}
	res := diffKeys(prior, next)
	if res.Removed != n || res.Added != n {
		t.Fatalf("removed=%d added=%d want %d/%d", res.Removed, res.Added, n, n)
	}
	// PackageChanges should have all entries (app layer does not truncate).
	if len(res.PackageChanges) != n*2 {
		t.Fatalf("changes=%d want %d", len(res.PackageChanges), n*2)
	}
}
