package templates

import (
	"embed"
	"html/template"
)

//go:embed mail/*.html
var mailFiles embed.FS

// VerifyEmailMail renders the verification email using HTML-aware escaping.
var VerifyEmailMail = template.Must(template.ParseFS(mailFiles, "mail/verify_email.html"))
