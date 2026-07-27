//go:build !windows

package transaction

import (
	"fmt"
	"os"
	"syscall"
)

func inodeVisitKey(info os.FileInfo) (string, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return "", false
	}
	return fmt.Sprintf("%d:%d", stat.Dev, stat.Ino), true
}

func createJunction(link, target string) error {
	return os.Symlink(target, link)
}
