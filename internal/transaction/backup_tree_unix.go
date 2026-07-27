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

func createJunction(link, substitute, print string) error {
	return os.Symlink(substitute, link)
}
