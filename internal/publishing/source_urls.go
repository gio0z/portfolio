package publishing

import (
	"fmt"
	"net/url"
	"strings"
)

// Source-url policy: source sites are follower-nominated third-party links kept
// as inert reference metadata. They are never fetched and never executed, so
// the only real hazard is letting a dangerous scheme reach an anchor href in a
// browser. Validation therefore restricts the scheme to http/https and requires
// an absolute URL with a host, which also rules out `javascript:`, `data:`,
// and `file:` payloads.
//
// URL reachability is deliberately NOT checked: a source site being offline
// must never block draft creation.

// NormalizeSourceURLs trims, validates, and de-duplicates source URLs while
// preserving the caller's order. An empty result is valid and means "no source
// site declared".
func NormalizeSourceURLs(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(raw))
	normalized := make([]string, 0, len(raw))
	for _, candidate := range raw {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return nil, NewValidationError(fmt.Sprintf("invalid source_urls entry %q: not a valid URL", trimmed), err)
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			return nil, NewValidationError(fmt.Sprintf(
				"invalid source_urls entry %q: scheme must be http or https", trimmed), nil)
		}
		if parsed.Host == "" {
			return nil, NewValidationError(fmt.Sprintf(
				"invalid source_urls entry %q: absolute URL with a host is required", trimmed), nil)
		}
		if _, duplicate := seen[trimmed]; duplicate {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil, nil
	}
	return normalized, nil
}
