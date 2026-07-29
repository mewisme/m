//go:build windows

package fsx_test

import (
	"os"
	"testing"

	"github.com/mewisme/mew/internal/fsx"
)

func TestGuardAncestorsJunctionEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	proj := root + `\proj`
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	mew := proj + `\.mew`
	if err := fsx.CreateMountPoint(mew, `\\?\`+outside, outside); err != nil {
		t.Skip("junction not supported:", err)
	}
	if err := fsx.GuardAncestors(proj, mew+`\txn`); err == nil {
		t.Fatal("expected junction rejection")
	}
}
