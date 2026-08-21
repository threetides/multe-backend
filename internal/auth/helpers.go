package auth

import (
	"net/url"
	"strings"
)

// validateReturnOrigin reduces a URL to its origin and checks it against the
// configured allowlist. Falls back to the first allowed origin.
func validateReturnOrigin(rawURL string) string {
	fallback := ""
	if len(AllowedReturnOrigins) > 0 {
		fallback = strings.TrimSpace(AllowedReturnOrigins[0])
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fallback
	}

	origin := u.Scheme + "://" + u.Host
	for _, allowed := range AllowedReturnOrigins {
		if origin == strings.TrimSpace(allowed) {
			return origin
		}
	}
	return fallback
}
