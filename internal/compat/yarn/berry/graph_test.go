package berry_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/compat/yarn/berry"
	"github.com/mewisme/mew/internal/testkit"
)

func TestBerryNMFixture(t *testing.T) {
	root := testkit.ModuleRoot(t)
	lockPath := filepath.Join(root, "fixtures", "locks", "yarn", "berry-nm", "yarn.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := berry.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	g, err := berry.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Packages) == 0 {
		t.Fatal("expected packages")
	}
}

func TestBerryPnPFixture(t *testing.T) {
	root := testkit.ModuleRoot(t)
	lockPath := filepath.Join(root, "fixtures", "locks", "yarn", "berry-pnp-readonly", "yarn.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := berry.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	g, ext, err := berry.ToPnPGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !berry.IsPnPGraph(ext) {
		t.Fatal("expected pnp linker tag")
	}
	if len(g.Packages) == 0 {
		t.Fatal("expected packages")
	}
}
