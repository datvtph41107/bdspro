package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/go-chi/chi/v5"
)

// chiPattern accepts Chi's native {name} and * wildcard patterns.
// Existing Gin-style :name and *filepath are translated at this isolated
// compatibility boundary; route declarations should use Chi syntax.
func chiPattern(pattern string) string {
	parts := strings.Split(pattern, "/")
	for i, part := range parts {
		if len(part) > 1 && part[0] == ':' {
			parts[i] = "{" + part[1:] + "}"
		} else if len(part) > 1 && part[0] == '*' {
			parts[i] = "*"
		}
	}
	return strings.Join(parts, "/")
}

func (r *Router) wrapChiIngress() http.Handler {
	// Chi owns BeforeRouting, matching and child router middleware.
	// This outer layer only enriches request metadata after an explicit
	// trusted-proxy check; no second matcher or middleware engine exists.
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.chiRouter.ServeHTTP(w, r.withTrustedClientIP(req))
	})
}

func (r *Router) setChiTrustedProxies(proxies []string) error {
	parsed := make([]netip.Prefix, 0, len(proxies))
	for _, raw := range proxies {
		if raw == "" {
			return fmt.Errorf("empty trusted proxy")
		}
		if prefix, err := netip.ParsePrefix(raw); err == nil {
			parsed = append(parsed, prefix.Masked())
			continue
		}
		addr, err := netip.ParseAddr(raw)
		if err != nil {
			return fmt.Errorf("invalid trusted proxy %q: %w", raw, err)
		}
		parsed = append(parsed, netip.PrefixFrom(addr, addr.BitLen()))
	}
	r.chiTrustedProxies = parsed
	return nil
}

type clientIPKey struct{}

// ClientIPFromRequest returns a verified IP, never an untrusted forwarded
// header. Middleware and typed contexts should use this instead of
// resolving X-Forwarded-For independently.
func ClientIPFromRequest(req *http.Request) string {
	if value, ok := req.Context().Value(clientIPKey{}).(string); ok && value != "" {
		return value
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}

func (r *Router) withTrustedClientIP(req *http.Request) *http.Request {
	raw, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		raw = req.RemoteAddr
	}
	peer, err := netip.ParseAddr(raw)
	if err != nil {
		return req
	}
	resolved := peer
	if r.chiIsTrusted(peer) {
		forwarded := strings.Split(req.Header.Get("X-Forwarded-For"), ",")
		// Walk right-to-left so an untrusted hop cannot forge a client IP.
		for i := len(forwarded) - 1; i >= 0; i-- {
			value := strings.TrimSpace(forwarded[i])
			if value == "" {
				continue
			}
			ip, parseErr := netip.ParseAddr(value)
			if parseErr != nil {
				break
			}
			resolved = ip
			if !r.chiIsTrusted(ip) {
				break
			}
		}
		if resolved == peer {
			if value := req.Header.Get("X-Real-IP"); value != "" {
				if ip, parseErr := netip.ParseAddr(value); parseErr == nil {
					resolved = ip
				}
			}
		}
	}
	return req.WithContext(context.WithValue(req.Context(), clientIPKey{}, resolved.String()))
}

func (r *Router) chiIsTrusted(ip netip.Addr) bool {
	for _, prefix := range r.chiTrustedProxies {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

// chi compile-time interface check: source's public surface is net/http.
var _ http.Handler = (*chi.Mux)(nil)
