package pnpm

import (
	"strings"
	"testing"
)

func TestDecodeRejectsDuplicateKeys(t *testing.T) {
	data := []byte(`lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      a: {specifier: 1, version: 1}
  .:
    devDependencies:
      b: {specifier: 2, version: 2}
packages: {}
snapshots: {}
`)
	_, err := Decode(data)
	if err == nil {
		t.Fatal("expected duplicate key error")
	}
	if !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("err=%v", err)
	}
}

func TestDecodeRejectsOversizeInput(t *testing.T) {
	data := make([]byte, maxLockBytes+1)
	_, err := Decode(data)
	if err == nil {
		t.Fatal("expected oversize error")
	}
}

func TestValidatePackageKeyRejectsLongPeerSuffix(t *testing.T) {
	key := "pkg@1.0.0(" + strings.Repeat("a", maxPeerSuffixLen+1) + ")"
	if err := validatePackageKey(key); err == nil {
		t.Fatal("expected peer suffix limit error")
	}
}

func TestNewPackageIndexRejectsOversize(t *testing.T) {
	keys := make([]string, maxIndexKeys+1)
	for i := range keys {
		keys[i] = "pkg@1.0." + strings.Repeat("0", 6) + string(rune('a'+i%26))
	}
	idx := NewPackageIndex(keys)
	_, err := ResolveDependencyTarget("pkg", "1.0.0", idx)
	if err == nil {
		t.Fatal("expected resolution failure on oversize index")
	}
}
