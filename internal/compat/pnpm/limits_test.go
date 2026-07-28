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
