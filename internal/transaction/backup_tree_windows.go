//go:build windows

package transaction

import (
	"os"

	"github.com/mewisme/mew/internal/fsx"
)

func inodeVisitKey(info os.FileInfo) (string, bool) {
	return "", false
}

func createJunction(link, substitute, print string) error {
	return fsx.CreateMountPoint(link, substitute, print)
}
