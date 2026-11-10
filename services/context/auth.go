package context

import (
	"net/http"
	"strings"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
)

// Authenticate verifies the signed first-party token but never grants session
// authority. The APIContext carries the untrusted-for-writes claims until the
// next admission step checks the current session against PostgreSQL.
func Authenticate() func(*APIContext) {
	return func(ctx *APIContext) {
		maker := authcrypto.MakerFromContext(ctx)
		parts := strings.Fields(ctx.CurrentRequest().Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			ctx.Header("WWW-Authenticate", "Bearer")
			ctx.APIError(http.StatusUnauthorized, "unauthenticated", "missing or invalid bearer token")
			return
		}
		payload, err := maker.VerifyToken(parts[1], authcrypto.TokenTypeAccess)
		if err != nil {
			ctx.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
			ctx.APIError(http.StatusUnauthorized, "invalid_access_token", "invalid access token")
			return
		}
		ctx.Doer = payload
		// IsSigned is set only by RequireSession after live DB verification.
	}
}

