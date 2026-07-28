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
	if err := pnpm.ValidateSupportedPnpm(doc); err == nil {
		t.Fatal("expected ValidateSupportedPnpm rejection")
	}
	_, err = pnpm.ToGraph(doc)
	if err == nil {
		t.Fatal("expected ToGraph rejection")
	}
}

func TestRejectUnsupportedFixtures(t *testing.T) {
	root := moduleRoot(t)
	cases := []struct {
		dir    string
		legacy bool
	}{
		{"v5.4", true},
		{"v7", true},
		{"v8", true},
		{"unknown-version", false},
		{"misleading-flat", false},
	}
	for _, tc := range cases {
		t.Run(tc.dir, func(t *testing.T) {
			path := filepath.Join(root, "fixtures", "locks", "pnpm", "unsupported", tc.dir, "pnpm-lock.yaml")
			prior, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = lockfile.DetectPnpm(prior)
			if err == nil {
				t.Fatal("expected detection rejection")
			}
			if apperr.CodeOf(err) != apperr.LockUnsupported {
				t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
			}
			_, err = pnpm.Adapter{}.Read(context.Background(), path)
			if err == nil {
				t.Fatal("expected adapter read rejection")
			}
			doc, err := pnpm.Decode(prior)
			if err != nil {
				t.Fatal(err)
			}
			if tc.legacy && !pnpm.IsLegacyUnsupported(doc) {
				t.Fatal("expected legacy classification")
			}
			if err := pnpm.ValidateSupportedPnpm(doc); err == nil {
				t.Fatal("expected ValidateSupportedPnpm rejection")
			}
		})
	}
}

func TestAdapterEncodeRejectsUnsupported(t *testing.T) {
	root := moduleRoot(t)
	path := filepath.Join(root, "fixtures", "locks", "pnpm", "unsupported", "v7", "pnpm-lock.yaml")
	prior, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pnpm.Adapter{}.EncodePreserving(context.Background(), path, nil, prior, nil, lockfile.Detection{})
	if err == nil {
		t.Fatal("expected encode rejection")
	}
	if apperr.CodeOf(err) != apperr.LockUnsupported {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}
