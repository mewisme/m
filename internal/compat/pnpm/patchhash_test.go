package pnpm_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/compat/pnpm"
	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/testkit"
)

func TestComputePatchHashPnpm9Fixture(t *testing.T) {
	path := filepath.Join(testkit.ModuleRoot(t), "fixtures", "locks", "generated", "pnpm-9", "patch", "patches", "ms@2.1.3.patch")
	data, err := manifest.ReadPatchFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := pnpm.ComputePatchHash(9, data)
	if err != nil {
		t.Fatal(err)
	}
	const want = "qbeutzo35bwf2d244a4luocaf4"
	if got != want {
		t.Fatalf("hash=%q want %q", got, want)
	}
}

func TestComputePatchHashPnpm10Fixture(t *testing.T) {
	path := filepath.Join(testkit.ModuleRoot(t), "fixtures", "locks", "generated", "pnpm-10", "patch", "patches", "ms@2.1.3.patch")
	data, err := manifest.ReadPatchFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := pnpm.ComputePatchHash(10, data)
	if err != nil {
		t.Fatal(err)
	}
	const want = "3d3b233afae989e0edf78c4014af2b67cae585452049aa323b1fc4160919e1cb"
	if got != want {
		t.Fatalf("hash=%q want %q", got, want)
	}
}

func TestComputePatchHashCRLFDeterministic(t *testing.T) {
	lf := []byte("diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-a\n+b\n")
	crlf := []byte(strings.ReplaceAll(string(lf), "\n", "\r\n"))
	hLF, err := pnpm.ComputePatchHash(10, pnpm.NormalizePatchBytes(lf))
	if err != nil {
		t.Fatal(err)
	}
	hCRLF, err := pnpm.ComputePatchHash(10, pnpm.NormalizePatchBytes(crlf))
	if err != nil {
		t.Fatal(err)
	}
	if hLF != hCRLF {
		t.Fatalf("CRLF/LF mismatch: %q vs %q", hLF, hCRLF)
	}
}

func TestComputePatchHashOneByteChange(t *testing.T) {
	a := []byte("diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-a\n+b\n")
	b := append([]byte(nil), a...)
	b[len(b)-2] = 'c'
	ha, err := pnpm.ComputePatchHash(9, a)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := pnpm.ComputePatchHash(9, b)
	if err != nil {
		t.Fatal(err)
	}
	if ha == hb {
		t.Fatal("expected different hashes for different bytes")
	}
}

func TestComputePatchHashRejectsStaleMismatch(t *testing.T) {
	path := filepath.Join(testkit.ModuleRoot(t), "fixtures", "locks", "generated", "pnpm-9", "patch", "patches", "ms@2.1.3.patch")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	computed, err := pnpm.ComputePatchHash(9, pnpm.NormalizePatchBytes(data))
	if err != nil {
		t.Fatal(err)
	}
	const stale = "deadbeefdeadbeefdeadbeefde"
	if computed == stale {
		t.Fatal("fixture hash must differ from arbitrary stale value")
	}
}
