package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const csrfProofHeader = "X-BDSPro-Request"

// browserSecurity is a browser boundary, not an authentication or authorization
// mechanism. CORS controls which origins can read credentialed responses; the
// unsafe-method gate is a separate, fail-closed CSRF mitigation.
func (server *Server) browserSecurity(ctx *gin.Context) {
	ctx.Writer.Header().Add("Vary", "Origin")
	origin := ctx.GetHeader("Origin")
	if origin != "" && server.allowedBrowserOrigin(ctx.Request, origin) {
		ctx.Header("Access-Control-Allow-Origin", origin)
		ctx.Header("Access-Control-Allow-Credentials", "true")
		ctx.Header("Access-Control-Expose-Headers", "X-Request-ID")
	}

	if ctx.Request.Method == http.MethodOptions && ctx.GetHeader("Access-Control-Request-Method") != "" {
		ctx.Writer.Header().Add("Vary", "Access-Control-Request-Method")
		ctx.Writer.Header().Add("Vary", "Access-Control-Request-Headers")
		if origin == "" || !server.allowedBrowserOrigin(ctx.Request, origin) ||
			!allowedPreflightMethod(ctx.GetHeader("Access-Control-Request-Method")) ||
			!allowedPreflightHeaders(ctx.GetHeader("Access-Control-Request-Headers")) {
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		}
		ctx.Header("Access-Control-Allow-Methods", "GET, HEAD, POST, DELETE, OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, X-BDSPro-Request")
		ctx.Header("Access-Control-Max-Age", "300")
		ctx.AbortWithStatus(http.StatusNoContent)
		return
	}

	switch ctx.Request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		ctx.Next()
		return
	}

	if strings.EqualFold(ctx.GetHeader("Sec-Fetch-Site"), "cross-site") {
		ctx.AbortWithStatus(http.StatusForbidden)
		return
	}
	// A present Origin must match the configured UI origin or the actual API
	// origin. The Host comparison uses the real request target, never a
	// spoofable X-Forwarded-Host or X-Forwarded-Proto.
	if origin != "" && !server.allowedBrowserOrigin(ctx.Request, origin) {
		ctx.AbortWithStatus(http.StatusForbidden)
		return
	}

	// Legacy token-free API requests are still accepted without browser
	// metadata. Cookie-authenticated writes and login are fail-closed.
	// The custom header is a browser non-simple header, requiring preflight
	// for cross-origin JavaScript; non-browser clients can set it directly.
	_, hasCookie := ctx.Request.Header["Cookie"]
	protected := hasCookie || ctx.FullPath() == "/auth/login" || ctx.FullPath() == "/auth/logout"
	if protected && origin == "" &&
		!strings.EqualFold(ctx.GetHeader("Sec-Fetch-Site"), "same-origin") &&
		ctx.GetHeader(csrfProofHeader) != "1" {
		ctx.AbortWithStatus(http.StatusForbidden)
		return
	}
	ctx.Next()
}

func (server *Server) allowedBrowserOrigin(req *http.Request, origin string) bool {
	if origin == "" || origin == "null" {
		return false
	}
	if server.allowedOrigin != "" && origin == server.allowedOrigin {
		return true
	}
	if req.Host == "" {
		return false
	}
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	return origin == scheme+"://"+req.Host
}

func allowedPreflightMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodDelete, http.MethodOptions:
		return true
	}
	return false
}

func allowedPreflightHeaders(headers string) bool {
	if headers == "" {
		return true
	}
	for _, part := range strings.Split(headers, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "content-type", "idempotency-key", "x-bdspro-request":
		default:
			return false
		}
	}
	return true
}
