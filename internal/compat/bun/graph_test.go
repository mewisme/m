package bun_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/compat/bun"
	_ "github.com/mewisme/mew/internal/compat/bun"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/testkit"
)

func TestBunFixtureV1Basic(t *testing.T) {
	root := testkit.ModuleRoot(t)
	lockPath := filepath.Join(root, "fixtures", "locks", "bun", "v1-basic", "bun.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := bun.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := bun.ValidateSupported(doc); err != nil {
		t.Fatal(err)
	}
	g, err := bun.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Packages) == 0 {
		t.Fatal("expected packages")
	}
	ext, ok := lockfile.ExtAdapterFor(project.IdentityBun)
	if !ok {
		t.Fatal("missing bun adapter")
	}
	g2, _, err := ext.ReadWithExtensions(context.Background(), lockPath)
	if err != nil {
		t.Fatal(err)
	}
	res, err := ext.(lockfile.PreservingEncoder).EncodePreserving(context.Background(), lockPath, g2, data, nil, bun.DetectFromDocument(doc))
	if err != nil {
		t.Fatal(err)
	}
	out := data
	if !res.Unchanged {
		out = res.Bytes
	}
	if !bytes.Equal(data, out) {
		t.Fatal("graph-equal no-op must preserve lock bytes")
	}
}

func TestBunRejectLockb(t *testing.T) {
	_, err := bun.Decode([]byte{0x00, 0x01, 0x02, 0x03})
	if err == nil {
		t.Fatal("expected binary rejection")
	}
}
