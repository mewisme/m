//go:build windows

package fsx_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/fsx"
)

func TestCreateMountPointRoundTrip(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "store")
	if err := os.MkdirAll(filepath.Join(target, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "vendor")
	out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
	if err != nil {
		t.Skip(err, string(out))
	}
	sub, pr, _, err := fsx.ReadMountPoint(link)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(link); err != nil {
		t.Fatal(err)
	}
	if err := fsx.CreateMountPoint(link, sub, pr); err != nil {
		t.Fatal(err)
	}
	if fsx.ReparseTag(link) != fsx.IOReparseTagMountPoint {
		t.Fatalf("tag 0x%X", fsx.ReparseTag(link))
	}
	if _, err := os.Stat(filepath.Join(link, "pkg")); err != nil {
		t.Fatal(err)
	}
}
