package setting

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

type ServerSettings struct {
	HTTPServerAddress   string
	BrowserPublicOrigin string
	TrustedProxies      []string
	AllowedOrigins      []string
}

var Server ServerSettings

func loadServerFrom(cfg ConfigProvider, runMode string) (ServerSettings, error) {
	sec := cfg.Section("server")
	result := ServerSettings{
		HTTPServerAddress:   sec.Key("HTTP_SERVER_ADDRESS").String(),
		BrowserPublicOrigin: sec.Key("PUBLIC_ORIGIN").String(),
	}
	if err := ValidateBrowserPublicOrigin(result.BrowserPublicOrigin, runMode); err != nil {
		return ServerSettings{}, err
	}
	peers, err := ParseTrustedProxies(sec.Key("TRUSTED_PROXIES").String())
	if err != nil {
		return ServerSettings{}, err
	}
	result.TrustedProxies = peers
	for _, origin := range strings.Split(cfg.Section("cors").Key("ALLOWED_ORIGINS").String(), ",") {
		if v := strings.TrimSpace(origin); v != "" {
			result.AllowedOrigins = append(result.AllowedOrigins, v)
		}
	}
	if err := validateListenerAddress(result.HTTPServerAddress); err != nil {
		return ServerSettings{}, err
	}
	return result, nil
}

func validateListenerAddress(address string) error {
	if address == "" || strings.TrimSpace(address) != address {
		return errors.New("server.HTTP_SERVER_ADDRESS is required without surrounding whitespace")
	}
	_, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("server.HTTP_SERVER_ADDRESS: %w", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("server.HTTP_SERVER_ADDRESS has invalid port %q", portText)
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
			return nil, errors.New("TRUSTED_PROXIES contains an empty entry")
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
				return errors.New("TRUSTED_PROXIES cannot include scoped IP zones")
			}
			continue
		}
		prefix, err := netip.ParsePrefix(peer)
		if err != nil {
			return fmt.Errorf("invalid TRUSTED_PROXIES CIDR/address %q: %w", peer, err)
		}
		if prefix.Bits() == 0 || (prefix.Addr().Is4In6() && prefix.Bits() <= 96) {
			return fmt.Errorf("TRUSTED_PROXIES must not trust entire IP address space: %q", peer)
		}
	}
	return nil
}

func ValidateBrowserPublicOrigin(raw, runMode string) error {
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Opaque != "" || u.Hostname() == "" ||
		u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" ||
		(u.Scheme != "http" && u.Scheme != "https") || u.String() != raw {
		return errors.New("server PUBLIC_ORIGIN must be a canonical HTTP(S) origin without path or credentials")
	}
	if u.Scheme == "http" {
		ip := net.ParseIP(u.Hostname())
		if runMode != "development" || (!strings.EqualFold(u.Hostname(), "localhost") && (ip == nil || !ip.IsLoopback())) {
			return errors.New("server PUBLIC_ORIGIN must use HTTPS except loopback development")
		}
	}
	return nil
}
