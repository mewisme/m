//go:build windows

package darkmode

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const regPath = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`

func IsDarkMode() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("darkmode: registry open %s: %w", regPath, err)
	}
	defer k.Close()

	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return false, fmt.Errorf("darkmode: registry get AppsUseLightTheme: %w", err)
	}

	// 0 = dark mode, 1 = light mode
	return v == 0, nil
}
