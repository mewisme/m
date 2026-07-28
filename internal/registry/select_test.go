package registry_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/registry"
	"github.com/mewisme/mew/internal/testkit"
)

func TestSelectMaxSatisfying(t *testing.T) {
	reg := testkit.LoadRegistry(t, "registry/v1")
	raw, err := os.ReadFile(filepath.Join(reg.Root, "packuments", "pkg-b.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := registry.ParsePackument(raw)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := registry.SelectMaxSatisfying(p, "^1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Version != "1.2.0" {
		t.Fatalf("got %s", meta.Version)
	}
	meta, err = registry.SelectMaxSatisfying(p, "latest")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Version != "1.2.0" {
		t.Fatalf("latest=%s", meta.Version)
	}
	_, err = registry.SelectMaxSatisfying(p, "^9.0.0")
	if apperr.CodeOf(err) != apperr.Resolve {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	vers := p.SortedVersionsSemver()
	if len(vers) < 2 || vers[0] != "1.0.0" || vers[len(vers)-1] != "2.0.0" {
		t.Fatalf("semver order: %v", vers)
	}
}
