package phonedelivery

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// TwilioConfig holds only transport credentials; application settings stay outside this package.
type TwilioConfig struct {
	AccountSID string
	AuthToken  string
	SenderID   string
}

// Constructors require a non-nil client with a bounded timeout and redirect policy.
// The composition root supplies one client shared by the configured adapters.
func NewTwilio(client *http.Client, cfg TwilioConfig) *Twilio {
	return &Twilio{client: client, cfg: cfg}
}
func NewZalo(client *http.Client, token, templateID, otpParam string) *Zalo {
	return &Zalo{client: client, token: token, templateID: templateID, otpParam: otpParam}
}
func NewTelegramGateway(client *http.Client, token string) *TelegramGateway {
	return &TelegramGateway{client: client, token: token}
}

type Twilio struct {
	client *http.Client
	cfg    TwilioConfig
}

func (s Twilio) Send(ctx context.Context, m Message) (Receipt, error) {
	form := url.Values{"To": {m.Phone}, "From": {s.cfg.SenderID},
		"Body": {"[" + s.cfg.SenderID + "] Your verification code is " + m.Code + ". It expires in 5 minutes."}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.twilio.com/2010-04-01/Accounts/"+s.cfg.AccountSID+"/Messages.json", strings.NewReader(form.Encode()))
	if err != nil {
		return Receipt{}, &Failure{Reason: "invalid_provider_endpoint"}
	}
	req.SetBasicAuth(s.cfg.AccountSID, s.cfg.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var response struct {
		SID  string `json:"sid"`
		Code int    `json:"code"`
	}
	status, err := exchange(s.client, req, &response)
	if err != nil {
		return Receipt{}, err
	}
	if status != http.StatusCreated || response.SID == "" {
		return Receipt{}, &Failure{Reason: "provider_rejected", HTTPStatus: status, ProviderCode: strconv.Itoa(response.Code)}
	}
	return Receipt{Provider: "twilio", MessageID: response.SID}, nil
}

type Zalo struct {
	client                      *http.Client
	token, templateID, otpParam string
}

func (s Zalo) Send(ctx context.Context, m Message) (Receipt, error) {
	req, err := jsonRequest(ctx, "https://business.openapi.zalo.me/message/template", map[string]any{
		"phone": strings.TrimPrefix(m.Phone, "+"), "template_id": s.templateID,
		"template_data": map[string]string{s.otpParam: m.Code},
		"tracking_id":   strings.ReplaceAll(uuid.NewString(), "-", ""),
	})
	if err != nil {
		return Receipt{}, err
	}
	req.Header.Set("access_token", s.token)
	var response struct {
		Error *int `json:"error"`
		Data  struct {
			ID string `json:"msg_id"`
		} `json:"data"`
	}
	status, err := exchange(s.client, req, &response)
	if err != nil {
		return Receipt{}, err
	}
	if status != http.StatusOK || response.Error == nil || *response.Error != 0 || response.Data.ID == "" {
		code := ""
		if response.Error != nil {
			code = strconv.Itoa(*response.Error)
		}
		return Receipt{}, &Failure{Reason: "provider_rejected", HTTPStatus: status, ProviderCode: code}
	}
	return Receipt{Provider: "zalo", MessageID: response.Data.ID}, nil
}

type TelegramGateway struct {
	client *http.Client
	token  string
}

func (s TelegramGateway) Send(ctx context.Context, m Message) (Receipt, error) {
	req, err := jsonRequest(ctx, "https://gatewayapi.telegram.org/sendVerificationMessage", map[string]any{
		"phone_number": m.Phone, "code": m.Code, "ttl": 300,
	})
	if err != nil {
		return Receipt{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	var response struct {
		OK     bool `json:"ok"`
		Result struct {
			ID string `json:"request_id"`
		} `json:"result"`
	}
	status, err := exchange(s.client, req, &response)
	if err != nil {
		return Receipt{}, err
	}
	if status != http.StatusOK || !response.OK || response.Result.ID == "" {
		return Receipt{}, &Failure{Reason: "provider_rejected", HTTPStatus: status}
	}
	return Receipt{Provider: "telegram_gateway", MessageID: response.Result.ID}, nil
}
