package auth

import (
	"context"
	"errors"
	auth_model "github.com/datvtph41107/bdspro/models/auth"
	"testing"

	"github.com/datvtph41107/bdspro/modules/setting"
)

func TestPhoneInputErrorsAreApplicationErrors(t *testing.T) {
	phone := PhoneOTP{Config: setting.Config{Environment: "development", OIDC: setting.OAuth2Provider{Issuer: "https://issuer.example.test"}}}
	ctx := context.Background()
	if err := phone.StartBind(ctx, 1, "", "bad-phone"); !errors.Is(err, auth_model.ErrInvalidPhone) {
		t.Fatalf("invalid phone: %v", err)
	}
	if err := phone.StartOAuth2Bind(ctx, 1, auth_model.OIDCClaims{Issuer: "https://issuer.example.test", Subject: "sub"}, "bad-phone"); !errors.Is(err, auth_model.ErrInvalidPhone) {
		t.Fatalf("invalid OAuth2 phone: %v", err)
	}
	if err := phone.VerifyBind(ctx, 1, "+84912345678", "123"); !errors.Is(err, auth_model.ErrOTPInvalid) {
		t.Fatalf("invalid code: %v", err)
	}
}
