//go:build !windows && !darwin && !linux

package darkmode

import "fmt"

func IsDarkMode() (bool, error) {
	return false, fmt.Errorf("%w: unsupported GOOS", ErrUnsupported)
}
