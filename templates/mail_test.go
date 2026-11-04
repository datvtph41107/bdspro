package templates

import (
	"bytes"
	"strings"
	"testing"
)

func TestVerifyEmailMailEscapesLink(t *testing.T) {
	var body bytes.Buffer
	if err := VerifyEmailMail.Execute(&body, struct{ VerificationURL string }{
		VerificationURL: "https://example.test/verify?email_id=1&secret_code=reused-secret",
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body.String(), "secret_code=reused-secret") ||
		!strings.Contains(body.String(), "&amp;") {
		t.Fatalf("verification link not HTML-escaped: %q", body.String())
	}
}

func TestVerifyEmailMailRejectsUnsafeScheme(t *testing.T) {
	var body bytes.Buffer
	if err := VerifyEmailMail.Execute(&body, struct{ VerificationURL string }{
		VerificationURL: "javascript:alert(1)",
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body.String(), "href=\"javascript:") {
		t.Fatalf("unsafe link scheme: %q", body.String())
	}
}
