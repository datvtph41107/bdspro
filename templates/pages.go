// Package templates owns static HTML assets rendered by HTTP handlers.
package templates

import (
	"embed"
	"html/template"
)

//go:embed auth/*.html
var pages embed.FS

// Pages are parsed exactly once, escaping all dynamic token fields.
var (
	VerifyEmailPage   = template.Must(template.ParseFS(pages, "auth/verify_email.html"))
	PasswordResetPage = template.Must(template.ParseFS(pages, "auth/password_reset.html"))
	OAuthLoginPage = template.Must(template.ParseFS(pages, "auth/oauth_login.html"))
	OAuthConsentPage = template.Must(template.ParseFS(pages, "auth/oauth_consent.html"))
)
