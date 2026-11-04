package setting

import (
	"fmt"
	"net/netip"
	"strings"
)

// ParseTrustedProxies reads the exact ingress peer addresses/CIDRs that Gin
// may trust for X-Forwarded-For. An empty setting intentionally trusts none.
func ParseTrustedProxies(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	peers := make([]string, 0, len(parts))
	for _, part := range parts {
		peer := strings.TrimSpace(part)
		if peer == "" {
			return nil, fmt.Errorf("TRUSTED_PROXIES contains an empty entry")
		}
		peers = append(peers, peer)
	}
	if err := ValidateTrustedProxies(peers); err != nil {
		return nil, err
	}
	return peers, nil
}

// ValidateTrustedProxies also checks manually constructed Config values.
// Do not trust the entire IPv4/IPv6 address space or accept hostnames/zones:
// the policy must identify network peers, not dynamic DNS names.
func ValidateTrustedProxies(peers []string) error {
	for _, peer := range peers {
		if peer == "" || strings.TrimSpace(peer) != peer {
			return fmt.Errorf("invalid TRUSTED_PROXIES entry %q", peer)
		}
		if ip, err := netip.ParseAddr(peer); err == nil {
			if ip.Zone() != "" {
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
