package presentation

import "os"

// EnvMap snapshots relevant environment variables for diagnostics and capability detection.
// Presentation env vars (MEW_OUTPUT, MEW_COLOR, etc.) are no longer read for presentation resolution.
func EnvMap() map[string]string {
	keys := []string{
		"MEW_DEBUG", "M_LOG", "NO_COLOR", "CLICOLOR", "CLICOLOR_FORCE", "CI",
		"GITHUB_ACTIONS", "TERM", "COLORTERM", "COLORFGBG", "TERM_PROGRAM", "WT_SESSION",
	}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			out[k] = v
		}
	}
	return out
}

// ConfigMap extracts presentation keys from a flat config string map.
// Presentation config keys (ui.output, ui.color, etc.) are no longer read.
func ConfigMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	keys := []string{
		"log.level",
	}
	out := make(map[string]string)
	for _, k := range keys {
		if v, ok := values[k]; ok {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
