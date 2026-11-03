package notification

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/datvtph41107/bdspro/modules/setting"
)

// NewPhoneCodeSender selects the configured phone delivery mechanism.
// This is notification transport configuration, not HTTP route registration.
// Development keeps the existing local-only behavior; production without
// an explicitly enabled provider intentionally has no phone sender.
func NewPhoneCodeSender(cfg setting.Config) (PhoneCodeSender, error) {
	if cfg.Environment == "development" {
		return Local{}, nil
	}

	switch cfg.PhoneDelivery {
	case "", "disabled":
		return nil, nil
	case "sms":
		endpoint, err := url.Parse(cfg.SMSGatewayURL)
		if err != nil || endpoint == nil || endpoint.Scheme != "https" || endpoint.Host == "" || cfg.SMSGatewayToken == "" {
			return nil, errors.New("SMS_GATEWAY_URL must use HTTPS and SMS_GATEWAY_TOKEN is required")
		}
		return SMSGateway{
			URL: cfg.SMSGatewayURL, Token: cfg.SMSGatewayToken,
			Client: &http.Client{Timeout: 8 * time.Second},
		}, nil
	case "zalo":
		if cfg.ZaloAccessToken == "" || cfg.ZaloTemplateID == "" || cfg.ZaloCodeField == "" {
			return nil, errors.New("Zalo ZBS delivery requires access token, template ID and code field")
		}
		return ZaloZBS{
			AccessToken: cfg.ZaloAccessToken, TemplateID: cfg.ZaloTemplateID,
			CodeField: cfg.ZaloCodeField, Client: &http.Client{Timeout: 8 * time.Second},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported PHONE_DELIVERY: %s", cfg.PhoneDelivery)
	}
}
