package conformance_test

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
)

func TestLockBridgeBunFixturesParse(t *testing.T) {
	root := moduleRoot(t)
	dir := filepath.Join(root, filepath.FromSlash("fixtures/locks/bun/v1-basic"))
	lockPath := filepath.Join(dir, "bun.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := bun.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	det := bun.DetectFromDocument(doc)
	if det.Format != bun.FormatV1 {
		t.Fatalf("det=%+v want format=%s", det, bun.FormatV1)
	}
	ext, ok := lockfile.ExtAdapterFor(project.IdentityBun)
	if !ok {
		t.Fatal("missing bun adapter")
	}
	g, _, err := ext.ReadWithExtensions(context.Background(), lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if g == nil || len(g.Packages) == 0 {
		t.Fatal("expected packages in graph")
	}
	before := append([]byte(nil), data...)
	res, err := ext.(lockfile.PreservingEncoder).EncodePreserving(context.Background(), lockPath, g, data, nil, det)
	if err != nil {
		t.Fatal(err)
	}
	out := before
	if !res.Unchanged {
		out = res.Bytes
	}
	if !bytes.Equal(before, out) {
		t.Fatal("graph-equal no-op must preserve lock bytes")
	}
}
