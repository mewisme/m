package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/tools/conformance/fixturemeta"
)

func TestVerifyFixturesValidTree(t *testing.T) {
	root, err := moduleRoot()
	if err != nil {
		t.Fatal(err)
	}
	gen := filepath.Join(root, "fixtures", "locks", "generated")
	dir := filepath.Join(gen, "pnpm-9", "basic")
	meta, err := fixturemeta.ReadMeta(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	if errs := meta.Validate(fixturemeta.ValidateOptions{
		WantProducer: "pnpm", WantProducerMajor: 9, WantFamily: "basic",
	}); len(errs) > 0 {
		t.Fatalf("validate: %v", errs)
	}
	if errs := fixturemeta.VerifyFixtureDir(dir, meta, "pnpm-lock.yaml"); len(errs) > 0 {
		t.Fatalf("verify dir: %v", errs)
	}
}

func TestVerifyFixturesCorruptLockHash(t *testing.T) {
	root, err := moduleRoot()
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "fixtures", "locks", "generated", "pnpm-9", "basic")
	tmp := t.TempDir()
	copyDir(t, src, tmp)
	metaPath := filepath.Join(tmp, "metadata.json")
	meta, err := fixturemeta.ReadMeta(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	meta.LockfileSha256 = "deadbeef"
	if errs := fixturemeta.VerifyFixtureDir(tmp, meta, "pnpm-lock.yaml"); len(errs) == 0 {
		t.Fatal("expected lock hash mismatch")
	}
}

func TestVerifyFixturesCorruptSourceDigest(t *testing.T) {
	root, err := moduleRoot()
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "fixtures", "locks", "sources", "pnpm", "basic")
	digest, err := fixturemeta.SourceTreeDigest(src)
	if err != nil {
		t.Fatal(err)
	}
	bad := digest + "00"
	if bad == digest {
		t.Fatal("bad digest unchanged")
	}
}

func TestVerifyFixturesPlaceholderCommandRejected(t *testing.T) {
	meta := fixturemeta.Meta{Command: "committed generated fixture placeholder"}
	if !fixturemeta.IsPlaceholderCommand(meta.Command) {
		t.Fatal("expected placeholder detection")
	}
	if errs := meta.Validate(fixturemeta.ValidateOptions{}); len(errs) == 0 {
		t.Fatal("expected validation errors for empty meta")
	}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		out := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
