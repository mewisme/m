package classic_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/compat/yarn/classic"
	"github.com/mewisme/mew/internal/testkit"
)

func TestClassicFixture(t *testing.T) {
	root := testkit.ModuleRoot(t)
	lockPath := filepath.Join(root, "fixtures", "locks", "yarn", "classic-v1", "yarn.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := classic.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	g, err := classic.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Packages) == 0 {
		t.Fatal("expected packages")
	}
	priorGraph, err := classic.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := classic.WriteGate(priorGraph, g, data); err != nil {
		t.Fatal(err)
	}
	_ = context.Background()
	_ = bytes.Equal(data, data)
}
