package types

import "net/http"

type PreMiddlewareProvider func(next http.Handler) http.Handler
