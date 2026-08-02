// Package darkmode detects OS-level dark/light mode preference.
// Platform-specific implementations of IsDarkMode live in build-tagged files.
package darkmode

import "errors"

// ErrUnsupported is returned when the platform has no detection mechanism.
var ErrUnsupported = errors.New("darkmode: unsupported platform")
