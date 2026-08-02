//go:build darwin

package darkmode

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

func IsDarkMode() (bool, error) {
	cmd := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return false, err
		}
		// Key not set = light mode
		return false, nil
	}
	return strings.TrimSpace(out.String()) == "Dark", nil
}
