package pnpm_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/compat/pnpm"
	"github.com/mewisme/mew/internal/lockfile"
	_ "github.com/mewisme/mew/internal/lockfile/mlock"
)

func TestRejectLegacyV6Flat(t *testing.T) {
	root := moduleRoot(t)
	path := filepath.Join(root, "fixtures", "locks", "pnpm", "unsupported", "v6", "pnpm-lock.yaml")
	prior, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = lockfile.DetectPnpm(prior)
	if err == nil {
		t.Fatal("expected legacy rejection")
	}
	if apperr.CodeOf(err) != apperr.LockUnsupported {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "5.4") || !strings.Contains(err.Error(), "regenerate") {
		t.Fatalf("err=%v", err)
	}

	_, err = pnpm.Adapter{}.Read(context.Background(), path)
	if err == nil {
		t.Fatal("expected adapter read rejection")
	}
	if apperr.CodeOf(err) != apperr.LockUnsupported {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestRejectLegacyV54Bytes(t *testing.T) {
	const src = `lockfileVersion: 5.4
dependencies:
  a:
    specifier: 1.0.0
    version: 1.0.0
packages:
  /a/1.0.0:
    resolution:
      integrity: sha512-test
`
	prior := []byte(src)
	doc, err := pnpm.Decode(prior)
	if err != nil {
		t.Fatal(err)
	}
	if !pnpm.IsLegacyUnsupported(doc) {
		t.Fatal("expected legacy layout")
	}
	_, err = pnpm.ToGraph(doc)
	if err == nil {
		t.Fatal("expected ToGraph rejection")
	}
}
