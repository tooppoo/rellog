package rellog

import (
	"fmt"
	"net/url"
	"strings"
)

func validateLink(raw string) []string {
	if raw == "" {
		return []string{"link URL must not be empty"}
	}
	if strings.TrimSpace(raw) == "" {
		return []string{"link URL must not be whitespace-only"}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return []string{fmt.Sprintf("link URL %q is invalid", raw)}
	}
	if !u.IsAbs() {
		return []string{fmt.Sprintf("link URL %q must be absolute", raw)}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return []string{fmt.Sprintf("link URL %q must use http or https", raw)}
	}
	if u.Host == "" {
		return []string{fmt.Sprintf("link URL %q must include a host", raw)}
	}
	return nil
}
