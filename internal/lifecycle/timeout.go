package lifecycle

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mewisme/m/internal/config"
)

const defaultScriptTimeout = 10 * time.Minute

// ScriptTimeout returns the lifecycle script execution timeout.
func ScriptTimeout(eff *config.Effective) time.Duration {
	if v := os.Getenv("MEW_LIFECYCLE_SCRIPT_TIMEOUT"); v != "" {
		if d, err := parseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	if eff != nil {
		if val, err := config.Get(eff, "lifecycle.script_timeout"); err == nil {
			switch raw := val.Raw.(type) {
			case string:
				if d, err := parseDuration(raw); err == nil && d > 0 {
					return d
				}
			case int:
				if raw > 0 {
					return time.Duration(raw) * time.Millisecond
				}
			case float64:
				if raw > 0 {
					return time.Duration(raw) * time.Millisecond
				}
			}
		}
	}
	return defaultScriptTimeout
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, strconv.ErrSyntax
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Duration(ms) * time.Millisecond, nil
	}
	return 0, strconv.ErrSyntax
}
