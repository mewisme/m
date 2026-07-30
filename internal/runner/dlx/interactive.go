package dlx

import (
	"io"
	"os"
)

// InteractivityDetector reports whether a reader is an interactive terminal.
type InteractivityDetector interface {
	IsInteractive(r io.Reader) bool
}

// DefaultInteractivityDetector uses os.File Stat and ModeCharDevice.
type DefaultInteractivityDetector struct{}

// IsInteractive returns true when r is a character device TTY.
func (DefaultInteractivityDetector) IsInteractive(r io.Reader) bool {
	f, ok := r.(interface {
		Stat() (os.FileInfo, error)
	})
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}
