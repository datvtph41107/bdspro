package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type ctxKey int

const accountKey ctxKey = 1

type Handler struct {
	S             *Service
	Development   bool
	SecureCookies bool
	AllowedOrigin string
}

func (h *Handler) cookieName() string {
	if h.SecureCookies {
		return "__Host-bdspro_session"
	}
	return "bdspro_session"
}
func (h *Handler) cookie(w http.ResponseWriter, token string, age int) {
	http.SetCookie(w, &http.Cookie{Name: h.cookieName(), Value: token, Path: "/", Secure: h.SecureCookies,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(dst)
}
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func (h *Handler) originAllowed(r *http.Request) bool {
	if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	} // non-browser clients; browser cookies need additional CSRF policy
	return h.AllowedOrigin != "" && origin == h.AllowedOrigin
}
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", func(w http.ResponseWriter, r *http.Request) {
		if !h.originAllowed(r) {
			http.Error(w, "forbidden origin", 403)
			return
		}
		var input struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if decode(w, r, &input) != nil {
			http.Error(w, "invalid input", 400)
			return
		}
		reg, err := h.S.Register(r.Context(), input.Email, input.Password)
		switch {
		case errors.Is(err, ErrInvalid):
			http.Error(w, "invalid input", 400)
			return
		case errors.Is(err, ErrEmailUsed):
			http.Error(w, "email unavailable", 409)
			return
		case errors.Is(err, ErrMailerUnavailable):
			http.Error(w, "verification delivery unavailable; contact support", 503)
			return
		case err != nil:
			http.Error(w, "registration failed", 500)
			return
		}
		out := map[string]any{"account_id": reg.AccountID, "verification_required": true}
		if h.Development {
			out["verification_token"] = reg.VerificationToken
		}
		reply(w, 201, out)
	})
	mux.HandleFunc("POST /auth/verify-email", func(w http.ResponseWriter, r *http.Request) {
		if !h.originAllowed(r) {
			http.Error(w, "forbidden origin", 403)
			return
		}
		var input struct {
			Token string `json:"token"`
		}
		if decode(w, r, &input) != nil {
			http.Error(w, "invalid input", 400)
			return
		}
		err := h.S.VerifyEmail(r.Context(), strings.TrimSpace(input.Token))
		if errors.Is(err, ErrInvalidToken) {
			http.Error(w, "invalid or expired token", 400)
			return
		}
		if err != nil {
			http.Error(w, "verification failed", 500)
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("POST /auth/login", func(w http.ResponseWriter, r *http.Request) {
		if !h.originAllowed(r) {
			http.Error(w, "forbidden origin", 403)
			return
		}
		var input struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if decode(w, r, &input) != nil {
			http.Error(w, "invalid input", 400)
			return
		}
		res, err := h.S.Login(r.Context(), input.Email, input.Password)
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			http.Error(w, "invalid credentials", 401)
			return
		case errors.Is(err, ErrUnverified):
			http.Error(w, "email verification required", 403)
			return
		case err != nil:
			http.Error(w, "login failed", 500)
			return
		}
		h.cookie(w, res.Token, int(time.Until(res.ExpiresAt).Seconds()))
		reply(w, 200, map[string]any{"account_id": res.AccountID})
	})
	mux.HandleFunc("GET /auth/me", func(w http.ResponseWriter, r *http.Request) {
		id, ok := h.Account(r)
		if !ok {
			http.Error(w, "unauthorized", 401)
			return
		}
		reply(w, 200, map[string]any{"account_id": id})
	})
	mux.HandleFunc("POST /auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if !h.originAllowed(r) {
			http.Error(w, "forbidden origin", 403)
			return
		}
		c, err := r.Cookie(h.cookieName())
		if err == nil {
			_ = h.S.Logout(r.Context(), c.Value)
		}
		h.cookie(w, "", -1)
		w.WriteHeader(204)
	})
}
func (h *Handler) Account(r *http.Request) (int64, bool) {
	c, err := r.Cookie(h.cookieName())
	if err != nil {
		return 0, false
	}
	id, err := h.S.AccountFromToken(r.Context(), c.Value)
	return id, err == nil && id > 0
}
func AccountContext(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, accountKey, id)
}
func AccountID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(accountKey).(int64)
	return id, ok && id > 0
}
func (h *Handler) RequireAccount(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := h.Account(r)
		if !ok {
			http.Error(w, "unauthorized", 401)
			return
		}
		next.ServeHTTP(w, r.WithContext(AccountContext(r.Context(), id)))
	})
}
