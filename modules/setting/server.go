package setting

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

var Server struct {
	HTTPServerAddress   string
	BrowserPublicOrigin string
	TrustedProxies      []string
	AllowedOrigins      []string
}

// loadServerFrom owns the HTTP listener and ingress configuration.
func loadServerFrom(rootCfg ConfigProvider) error {
	sec := rootCfg.Section("server")
	Server.HTTPServerAddress = sec.Key("HTTP_SERVER_ADDRESS").String()
	Server.BrowserPublicOrigin = sec.Key("PUBLIC_ORIGIN").String()
	if err := ValidateBrowserPublicOrigin(Server.BrowserPublicOrigin, Common.RunMode); err != nil {
		return err
	}
	peers, err := ParseTrustedProxies(sec.Key("TRUSTED_PROXIES").String())
	if err != nil {
		return err
	}
	Server.TrustedProxies = peers
	Server.AllowedOrigins = Server.AllowedOrigins[:0]
	for _, origin := range strings.Split(rootCfg.Section("cors").Key("ALLOWED_ORIGINS").String(), ",") {
		if value := strings.TrimSpace(origin); value != "" {
			Server.AllowedOrigins = append(Server.AllowedOrigins, value)
		}
	}
	if Server.HTTPServerAddress == "" {
		return fmt.Errorf("server.HTTP_SERVER_ADDRESS is required")
	}
	return nil
}

func ParseTrustedProxies(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	peers := strings.Split(raw, ",")
	for i := range peers {
		peers[i] = strings.TrimSpace(peers[i])
		if peers[i] == "" {
			return nil, fmt.Errorf("TRUSTED_PROXIES contains an empty entry")
		}
	}
	if err := ValidateTrustedProxies(peers); err != nil {
		return nil, err
	}
	return peers, nil
}

func ValidateTrustedProxies(peers []string) error {
	for _, peer := range peers {
		if peer == "" || strings.TrimSpace(peer) != peer {
			return fmt.Errorf("invalid TRUSTED_PROXIES entry %q", peer)
		}
		if addr, err := netip.ParseAddr(peer); err == nil {
			if addr.Zone() != "" {
				return fmt.Errorf("TRUSTED_PROXIES cannot include scoped IP zones")
			}
			continue
		}
		prefix, err := netip.ParsePrefix(peer)
		if err != nil {
			return fmt.Errorf("invalid TRUSTED_PROXIES CIDR/address %q: %w", peer, err)
		}
		if prefix.Bits() == 0 || (prefix.Addr().Is4In6() && prefix.Bits() <= 96) {
			return fmt.Errorf("TRUSTED_PROXIES must not trust the entire IP address space: %q", peer)
		}
	}
	return nil
}

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
			(!strings.EqualFold(u.Hostname(), "localhost") && (ip == nil || !ip.IsLoopback())) {
			return fmt.Errorf("server PUBLIC_ORIGIN must use HTTPS except loopback development")
		}
	}
	return nil
}
