package phonedelivery

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
