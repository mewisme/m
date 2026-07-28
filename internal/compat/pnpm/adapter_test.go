package pnpm_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/compat/nub"
	"github.com/mewisme/mew/internal/compat/pnpm"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
	_ "github.com/mewisme/mew/internal/lockfile/mlock"
)

func moduleRoot(t *testing.T) string {
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

func TestGoldenRoundTrip(t *testing.T) {
	root := filepath.Join(moduleRoot(t), "fixtures", "locks", "pnpm")
	cases := []struct {
		gen   string
		major int
	}{
		{"v6", 0},
		{"v9", 9},
		{"v10", 10},
		{"v11", 11},
	}
	for _, tc := range cases {
		t.Run(tc.gen, func(t *testing.T) {
			path := filepath.Join(root, tc.gen, "pnpm-lock.yaml")
			prior, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := pnpm.Decode(prior)
			if err != nil {
				t.Fatal(err)
			}
			g, err := pnpm.ToGraph(doc)
			if err != nil {
				t.Fatal(err)
			}
			det, err := lockfile.DetectPnpmWithMajor(prior, tc.major)
			if err != nil {
				t.Fatal(err)
			}
			if tc.major != 0 {
				det.ExplicitMajor = true
			}
			res, err := pnpm.Adapter{}.EncodePreserving(context.Background(), path, g, prior, doc.Extensions, det)
			if err != nil {
				t.Fatal(err)
			}
			if !res.Unchanged {
				t.Fatalf("%s: expected unchanged graph", tc.gen)
			}
			if !bytes.Equal(res.Bytes, prior) {
				t.Fatalf("%s: byte mismatch on unchanged graph", tc.gen)
			}
		})
	}
}

func TestAmbiguousWriteFailsClosed(t *testing.T) {
	path := filepath.Join(moduleRoot(t), "fixtures", "locks", "pnpm", "v9", "pnpm-lock.yaml")
	prior, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := pnpm.Decode(prior)
	if err != nil {
		t.Fatal(err)
	}
	g, err := pnpm.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	g.Packages = append(g.Packages, graph.Package{ID: graph.PackageID{Name: "chalk", Version: "5.0.0"}})
	det, err := lockfile.DetectPnpm(prior)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pnpm.Adapter{}.EncodePreserving(context.Background(), path, g, prior, nil, det)
	if err == nil {
		t.Fatal("expected ambiguous write failure")
	}
	if apperr.CodeOf(err) != apperr.LockAmbiguous {
		t.Fatalf("code=%s want %s", apperr.CodeOf(err), apperr.LockAmbiguous)
	}
}

func TestExplicitMajorAllowsWrite(t *testing.T) {
	path := filepath.Join(moduleRoot(t), "fixtures", "locks", "pnpm", "v9", "pnpm-lock.yaml")
	prior, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := pnpm.Decode(prior)
	if err != nil {
		t.Fatal(err)
	}
	g, err := pnpm.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	det, err := lockfile.DetectPnpmWithMajor(prior, 9)
	if err != nil {
		t.Fatal(err)
	}
	det.ExplicitMajor = true
	res, err := pnpm.Adapter{}.EncodePreserving(context.Background(), path, g, prior, doc.Extensions, det)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Unchanged {
		t.Fatal("expected unchanged")
	}
}

func TestNubRoundTrip(t *testing.T) {
	path := filepath.Join(moduleRoot(t), "fixtures", "locks", "nub", "v1-basic", "nub.lock")
	prior, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	g, ext, err := nub.Adapter{}.ReadWithExtensions(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := nub.Adapter{}.EncodePreserving(context.Background(), path, g, prior, ext, lockfile.Detection{Format: "nub", Confidence: lockfile.DetectionCertain})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Unchanged || !bytes.Equal(res.Bytes, prior) {
		t.Fatal("expected byte-identical nub.lock")
	}
}
