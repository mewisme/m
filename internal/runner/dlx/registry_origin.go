package dlx

import (
	"net/url"
	"strings"
)

// SanitizeRegistryOrigin returns a stable registry origin key without credentials.
func SanitizeRegistryOrigin(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "default"
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return strings.ToLower(strings.TrimRight(baseURL, "/"))
	}
	u.User = nil
	u.Fragment = ""
	u.RawQuery = ""
	scheme := strings.ToLower(u.Scheme)
	if scheme == "" {
		scheme = "https"
	}
	host := strings.ToLower(u.Host)
	return scheme + "://" + host + strings.TrimRight(u.Path, "/")
}
