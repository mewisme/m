//go:build windows

package transaction_test

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"

	"github.com/mewisme/m/internal/transaction"
)

var (
	modKernel32            = syscall.NewLazyDLL("kernel32.dll")
	procCreateSymbolicLink = modKernel32.NewProc("CreateSymbolicLinkW")
)

const symbolicLinkFlagDirectory = 0x1

func junctionDir(target, link string) error {
	targetPtr, err := syscall.UTF16PtrFromString(`\\?\` + target)
	if err != nil {
		return err
	}
	linkPtr, err := syscall.UTF16PtrFromString(link)
	if err != nil {
		return err
	}
	r, _, e := procCreateSymbolicLink.Call(
		uintptr(unsafe.Pointer(linkPtr)),
		uintptr(unsafe.Pointer(targetPtr)),
		uintptr(symbolicLinkFlagDirectory),
	)
	if r == 0 {
		if e != syscall.Errno(0) {
			return e
		}
		return syscall.EINVAL
	}
	return nil
}

func syscallMkfifo(path string, mode uint32) error {
	return syscall.EINVAL
}

func TestBackupTreeJunctionRoundTrip(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "store")
	if err := os.MkdirAll(filepath.Join(target, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "vendor")
	if err := junctionDir(target, link); err != nil {
		t.Skip("junction not supported:", err)
	}

	txn := transaction.NewRunner(root)
	ctx := context.Background()
	if err := txn.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	if err := txn.RecordBackup("vendor"); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(link, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := txn.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(link, "pkg")); err != nil {
		t.Fatalf("junction not restored: %v", err)
	}
}
