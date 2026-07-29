package conformance_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/compat/npm"
	_ "github.com/mewisme/mew/internal/compat/npm"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
)

func TestLockBridgeNpmFixturesParse(t *testing.T) {
	cases := []struct {
		rel    string
		format string
		major  int
	}{
		{"fixtures/locks/npm/v2-basic", npm.FormatV2, 2},
		{"fixtures/locks/npm/v3-workspaces", npm.FormatV3, 3},
	}
	root := moduleRoot(t)
	for _, tc := range cases {
		t.Run(tc.rel, func(t *testing.T) {
			dir := filepath.Join(root, filepath.FromSlash(tc.rel))
			lockPath := filepath.Join(dir, "package-lock.json")
			data, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := npm.Decode(data)
			if err != nil {
				t.Fatal(err)
			}
			det := npm.DetectFromDocument(doc)
			if det.Format != tc.format || det.ProducerMajor != tc.major {
				t.Fatalf("det=%+v want format=%s major=%d", det, tc.format, tc.major)
			}
			ext, ok := lockfile.ExtAdapterFor(project.IdentityNPM)
			if !ok {
				t.Fatal("missing npm adapter")
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
		})
	}
}

func TestLockBridgeNpmRejectV1(t *testing.T) {
	root := moduleRoot(t)
	path := filepath.Join(root, "testdata", "lockfile", "npm-roundtrip", "v1-reject.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = npm.Decode(data)
	if err == nil {
		t.Fatal("expected v1 rejection")
	}
}
