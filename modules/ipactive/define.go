package ipactive

import (
	"errors"
	"strings"
	"time"
)

type Action string

const (
	Register                Action = "register"
	Login                   Action = "login"
	VerifyEmail             Action = "verify_email"
	ResendVerificationEmail Action = "verify_email_resend"
	RenewAccess             Action = "refresh"
	RevokeToken             Action = "revoke"
	RequestPasswordReset    Action = "password_reset"
	ConfirmPasswordReset    Action = "password_reset_confirm"
	StartOAuth2Login        Action = "oauth2_login_start"
	StartPhoneRegister      Action = "phone_register_start"
	VerifyPhoneRegister     Action = "phone_register_verify"
	StartPhoneLogin         Action = "phone_login_start"
	VerifyPhoneLogin        Action = "phone_login_verify"
	StartEmailLink          Action = "email_link_start"
	ConfirmEmailLink        Action = "email_link_verify"
	StartOAuth2Link         Action = "oauth2_link_start"
	StartOAuth2PhoneBind    Action = "phone_oauth2_bind_start"
	AcceptTeamInvite        Action = "team_invite_accept"
	CreateTeamInvite        Action = "team_invite_issue"
	CreateTeam              Action = "team_create"
	RenameTeam              Action = "team_rename"
	StartPhoneBind          Action = "phone_bind_start"
	VerifyPhoneBind         Action = "phone_bind_verify"
	SMSDestinationCooldown  Action = "sms_destination_cooldown"
	SMSDestinationHour      Action = "sms_destination_hour"
	SMSDestinationDay       Action = "sms_destination_day"
	SMSGlobalDay            Action = "sms_global_day"
	SMSVerifyDestination    Action = "sms_verify_destination"
)

type Mode string

const (
	ModeCount Mode = "count"

	ModeCooldown Mode = "cooldown"
)

type Policy struct {
	Mode   Mode
	Limit  int64
	Window time.Duration
}

func (p Policy) Validate() error {
	if p.Window <= 0 {
		return errors.New("IPActive window must be positive")
	}
	switch p.Mode {
	case ModeCount:
		if p.Limit <= 0 {
			return errors.New("IPActive count limit must be positive")
		}
	case ModeCooldown:
		if p.Limit != 1 {
			return errors.New("IPActive cooldown must have limit 1")
		}
	default:
		return errors.New("unsupported IPActive mode")
	}
	return nil
}

func validAction(action Action) bool {
	if action == "" || len(action) > 64 {
		return false
	}
	return strings.IndexFunc(string(action), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_')
	}) == -1
}

func DefaultPolicies() map[Action]Policy {
	count := func(limit int64, window time.Duration) Policy {
		return Policy{Mode: ModeCount, Limit: limit, Window: window}
	}
	return map[Action]Policy{
		Register:                count(10, time.Hour),
		Login:                   count(30, time.Minute),
		VerifyEmail:             count(60, time.Minute),
		ResendVerificationEmail: count(10, time.Hour),
		RenewAccess:             count(60, time.Minute),
		RevokeToken:             count(60, time.Minute),
		RequestPasswordReset:    count(10, time.Hour),
		ConfirmPasswordReset:    count(20, time.Hour),
		StartOAuth2Login:        count(20, time.Minute),
		StartPhoneRegister:      count(10, time.Hour),
		VerifyPhoneRegister:     count(40, 10*time.Minute),
		StartPhoneLogin:         count(30, time.Minute),
		VerifyPhoneLogin:        count(40, 10*time.Minute),
		StartEmailLink:          count(10, time.Hour),
		ConfirmEmailLink:        count(20, time.Hour),
		StartOAuth2Link:         count(10, time.Minute),
		StartOAuth2PhoneBind:    count(10, time.Hour),
		AcceptTeamInvite:        count(40, 10*time.Minute),
		CreateTeamInvite:        count(10, time.Hour),
		CreateTeam:              count(30, time.Hour),
		RenameTeam:              count(120, time.Hour),
		StartPhoneBind:          count(10, time.Hour),
		VerifyPhoneBind:         count(40, 10*time.Minute),
		SMSDestinationCooldown:  {Mode: ModeCooldown, Limit: 1, Window: time.Minute},
		SMSDestinationHour:      count(5, time.Hour),
		SMSDestinationDay:       count(10, 24*time.Hour),
		SMSGlobalDay:            count(100, 24*time.Hour),
		SMSVerifyDestination:    count(20, 15*time.Minute),
	}
}
