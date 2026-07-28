package lifecycle

import (
	"strconv"
	"strings"
	"time"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
)

const defaultScriptTimeout = 10 * time.Minute

// ScriptTimeout returns the lifecycle script execution timeout from effective config.
func ScriptTimeout(eff *config.Effective) (time.Duration, error) {
	if eff != nil {
		if val, err := config.Get(eff, "lifecycle.script_timeout"); err == nil {
			switch raw := val.Raw.(type) {
			case string:
				if strings.TrimSpace(raw) == "" {
					return defaultScriptTimeout, nil
				}
				d, err := parseDuration(raw)
				if err != nil {
					return 0, apperr.Wrap(apperr.Config, "lifecycle.timeout", "lifecycle.script_timeout", err)
				}
				if d <= 0 {
					return 0, apperr.New(apperr.Config, "lifecycle.timeout", "lifecycle.script_timeout", "timeout must be positive")
				}
				return d, nil
			case int:
				if raw <= 0 {
					return 0, apperr.New(apperr.Config, "lifecycle.timeout", "lifecycle.script_timeout", "timeout must be positive")
				}
				return time.Duration(raw) * time.Millisecond, nil
			case float64:
				if raw <= 0 {
					return 0, apperr.New(apperr.Config, "lifecycle.timeout", "lifecycle.script_timeout", "timeout must be positive")
				}
				return time.Duration(raw) * time.Millisecond, nil
			}
		}
	}
	return defaultScriptTimeout, nil
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
