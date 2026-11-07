package setting

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateBrowserPublicOrigin checks the server's configured, canonical origin.
// Its absence disables the browser OAuth surface without affecting API OAuth.
func ValidateBrowserPublicOrigin(raw, environment string) error {
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Opaque != "" || u.Hostname() == "" ||
		u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" ||
		(u.Scheme != "http" && u.Scheme != "https") || u.String() != raw {
		return fmt.Errorf("server PUBLIC_ORIGIN must be a canonical HTTP(S) origin without path or credentials")
	}
	if u.Scheme == "http" {
		ip := net.ParseIP(u.Hostname())
		if environment != "development" ||
			!strings.EqualFold(u.Hostname(), "localhost") && (ip == nil || !ip.IsLoopback()) {
			return fmt.Errorf("server PUBLIC_ORIGIN must use HTTPS except loopback development")
		}
	}
	return nil
}
