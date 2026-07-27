//go:build !windows

package linker

import "github.com/mewisme/m/internal/apperr"

func junctionDir(src, dest string) error {
	return apperr.New(apperr.IO, "linker.apply", "junction", "unsupported")
}
