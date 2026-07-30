//go:build !windows

package fsx

import (
	"context"
	"os"
)

func renamePath(ctx context.Context, src, dst string) error {
	_ = ctx
	return os.Rename(src, dst)
}

func isTransientRenameErr(err error) bool {
	return false
}
