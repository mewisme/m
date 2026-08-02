//go:build linux

package darkmode

import (
	"bytes"
	"os/exec"
	"strings"
)

func IsDarkMode() (bool, error) {
	// Try gsettings (GNOME / XDG desktop portal compatible).
	cmd := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false, ErrUnsupported
	}
	// Output is e.g. 'prefer-dark' or 'default' (single-quoted).
	v := strings.TrimSpace(out.String())
	v = strings.Trim(v, "'")
	return v == "prefer-dark", nil
}
