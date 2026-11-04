# Ingress source-IP ownership: trusted reverse proxies

BDSPro uses Gin's `ClientIP()` in IPActive, credential throttling and request
monitoring. Trusting a forged forwarded header therefore changes authentication
admission, not only logs.

## Deployment contract

- Empty `[server] TRUSTED_PROXIES` means **trust no X-Forwarded-For**:
  Gin resolves the direct remote TCP peer instead.
- Configure only the **immediate ingress reverse-proxy peers** as numeric IPs
  or CIDRs, separated by commas. Example:
  `TRUSTED_PROXIES = 10.0.5.12,10.0.6.0/28`.
- Config is read from `app.ini` and may be overridden in memory by
  `BDS__SERVER__TRUSTED_PROXIES`. The value is validated during settings load,
  and direct programmatic construction of NewServer is validated as well.
- Invalid peers, empty list items and catch-all IPv4/IPv6 ranges fail startup.
  No dynamic hostnames, IPv6 scoped zones or silent fallback to trusting
  every proxy.
- Use ingress network ACLs: if an attacker can connect *as a trusted peer*,
  spoofing X-Forwarded-For is still possible. Review any upstream proxy chain
  and header rewriting at the actual deployment.

`NormalRoutes` defaults to no proxy trust for route-registration and standalone
tests. The production `NewServer` configures the validated peers on the
constructed Gin engine before exposing its HTTP handler. No second IP parser
is added to middleware; all consumers continue using `c.ClientIP()`.

## Acceptance

Settings tests prove INI override and dangerous-list rejection. Gin tests
exercise forged forwarded headers for direct/untrusted TCP peers and accepted
headers for explicitly trusted peers. No IPActive Redis key, auth status,
public API endpoint or database schema change is required.
