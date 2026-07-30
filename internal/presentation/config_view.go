package presentation

import "os"

// EnvMap snapshots relevant environment variables for resolution.
func EnvMap() map[string]string {
	keys := []string{
		"MEW_OUTPUT", "MEW_LOG_FORMAT", "MEW_COLOR", "MEW_PROGRESS", "MEW_UNICODE",
		"MEW_INTERACTIVE", "MEW_ACCESSIBLE", "MEW_LOG_LEVEL", "MEW_PRESENTATION",
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
func ConfigMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	keys := []string{
		"ui.output", "ui.color", "ui.progress", "ui.unicode", "ui.interactive",
		"ui.accessible", "ui.summary", "ui.theme", "log.level",
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
