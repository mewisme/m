package config

import (
	"time"

	"github.com/mewisme/mew/internal/apperr"
)

// RedactedPlaceholder is the single rendering for a suppressed secret value.
// Every output path uses this constant so JSON, rich, and plain output can
// never disagree about whether a value was withheld.
const RedactedPlaceholder = "<redacted>"

// IsSecret reports whether key is declared secret by the registry.
// Unknown keys are not secret: the registry is the only authority.
func IsSecret(key string) bool {
	canon := key
	if c := CanonicalKey(key); c != "" {
		canon = c
	}
	spec := KeySpec(canon)
	return spec != nil && spec.Secret
}

// RedactValue returns the value to display for key. Secret keys always render
// as RedactedPlaceholder, including in structured output — a machine-readable
// format is not a reason to leak. Non-secret values pass through unchanged.
//
// Call this at the output boundary, once, rather than redacting at each call
// site: one helper means one place to audit.
func RedactValue(key string, raw any) any {
	if IsSecret(key) {
		return RedactedPlaceholder
	}
	return raw
}

// RedactString is RedactValue for already-formatted display strings.
func RedactString(key, formatted string) string {
	if IsSecret(key) {
		return RedactedPlaceholder
	}
	return formatted
}

// DurationValue returns the typed duration for a duration-typed key.
//
// Unlike Duration, which silently substitutes a default, this reports the
// failure: an unknown key, a key whose schema type is not duration, or a
// stored value that does not parse. Callers that must not proceed on a
// misconfigured timeout use this; callers with a meaningful fallback use
// Duration.
func DurationValue(eff *Effective, key string) (time.Duration, error) {
	canon := key
	if c := CanonicalKey(key); c != "" {
		canon = c
	}
	spec := KeySpec(canon)
	if spec == nil {
		return 0, apperr.New(apperr.Config, "config.duration", key, "unknown key")
	}
	if spec.Type != TypeDuration {
		return 0, apperr.New(apperr.Config, "config.duration", key,
			"key is "+string(spec.Type)+", not duration")
	}
	v, err := Get(eff, canon)
	if err != nil {
		return 0, err
	}
	s, ok := v.Raw.(string)
	if !ok {
		return 0, apperr.New(apperr.Config, "config.duration", key, "value is not a duration string")
	}
	d, perr := ParseDuration(s)
	if perr != nil {
		return 0, apperr.Wrap(apperr.Config, "config.duration", key, perr)
	}
	return d, nil
}
