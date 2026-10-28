package service

import (
	"context"
	"errors"
	"testing"

	apperrors "github.com/datvtph41107/bdspro/internal/errors"
	"github.com/datvtph41107/bdspro/internal/config"
)

func TestPhoneInputErrorsAreApplicationErrors(t *testing.T) {
	phone := PhoneOTP{Config: config.Config{Environment: "development"}}
	ctx := context.Background()
	if err := phone.StartBind(ctx, 1, "", "bad-phone"); !errors.Is(err, apperrors.InvalidPhone) {
		t.Fatalf("invalid phone: %v", err)
	}
	if err := phone.StartGoogleBind(ctx, 1, "sub", "bad-phone"); !errors.Is(err, apperrors.InvalidPhone) {
		t.Fatalf("invalid Google phone: %v", err)
	}
	if err := phone.VerifyBind(ctx, 1, "+84912345678", "123"); !errors.Is(err, apperrors.OTPInvalid) {
		t.Fatalf("invalid code: %v", err)
	}
}
