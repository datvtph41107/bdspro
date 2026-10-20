package api

import (
	"crypto/sha256"
	"fmt"
	"math"
	"net"
	"net/netip"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// These are process-local admission limits for the current one-instance dev
// server. They are not a distributed, durable per-account lockout mechanism.
type ratePolicy struct {
	burst  int
	window time.Duration
}

type authBucket struct {
	tokens float64
	last   time.Time
	window time.Duration
}

type authLimiter struct {
	mu      sync.Mutex
	buckets map[string]authBucket
	maxKeys int
}

func newAuthLimiter(maxKeys int) *authLimiter {
	return &authLimiter{buckets: make(map[string]authBucket), maxKeys: maxKeys}
}

func fingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum[:])
}

// connectionIP uses the socket peer only, not X-Forwarded-For, X-Real-IP,
// Forwarded or Gin's proxy-derived ClientIP. Behind a reverse proxy this is
// the proxy's address until an explicit trusted-edge contract is established.
func connectionIP(remote string) string {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return "unknown"
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return "unknown"
	}
	return ip.Unmap().String()
}

// allow uses a token bucket: burst tokens, replenished evenly over window.
// All decisions and capacity checks are guarded by one mutex. An exhausted
// map fails closed instead of growing without bounds under rotating IPs/IDs.
func (l *authLimiter) allow(scope, identity string, policy ratePolicy, now time.Time) (bool, int) {
	key := scope + ":" + identity
	l.mu.Lock()
	defer l.mu.Unlock()

	b, found := l.buckets[key]
	if !found {
		if len(l.buckets) >= l.maxKeys {
			for k, previous := range l.buckets {
				if now.Sub(previous.last) >= previous.window {
					delete(l.buckets, k)
				}
			}
			if len(l.buckets) >= l.maxKeys {
				return false, 60
			}
		}
		b = authBucket{tokens: float64(policy.burst), last: now, window: policy.window}
	}
	if elapsed := now.Sub(b.last); elapsed > 0 {
		b.tokens = math.Min(float64(policy.burst),
			b.tokens+elapsed.Seconds()*float64(policy.burst)/policy.window.Seconds())
		b.last = now
	}
	if b.tokens < 1 {
		l.buckets[key] = b
		wait := int(math.Ceil((1-b.tokens)*policy.window.Seconds()/float64(policy.burst)))
		if wait < 1 {
			wait = 1
		}
		return false, wait
	}
	b.tokens--
	l.buckets[key] = b
	return true, 0
}

func authRateLimitExceeded(ctx *gin.Context, retrySeconds int) {
	ctx.Header("Retry-After", strconv.Itoa(retrySeconds))
	writeAPIError(ctx, http.StatusTooManyRequests, "too many requests")
}

// limitAuthClient executes before JSON binding, password hashing and DB/SMTP
// calls. Route policy is fixed server-side; do not key on query parameters.
func (server *Server) limitAuthClient(route string, policy ratePolicy) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ip := fingerprint(connectionIP(ctx.Request.RemoteAddr))
		ok, wait := server.authLimiter.allow("ip-"+route, ip, policy, time.Now())
		if !ok {
			authRateLimitExceeded(ctx, wait)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
