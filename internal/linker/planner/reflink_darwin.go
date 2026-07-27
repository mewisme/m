//go:build darwin

package planner

import "errors"

// ponytail: clonefile deferred; probe falls back to hardlink/copy.
func platformReflink(_, _ string) error {
	return errors.New("reflink unsupported")
}
