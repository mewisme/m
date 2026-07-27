//go:build windows

package transaction

import (
	"os"

	"github.com/mewisme/m/internal/fsx"
)

func inodeVisitKey(info os.FileInfo) (string, bool) {
	return "", false
}

func createJunction(link, substitute, print string) error {
	return fsx.CreateMountPoint(link, substitute, print)
}
