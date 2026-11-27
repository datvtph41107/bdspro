package phonedelivery

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type twilioSender struct {
	client     *http.Client
	accountSID string
	authToken  string
	senderID   string
}

func (s twilioSender) Send(ctx context.Context, m message) (receipt, error) {
	form := url.Values{"To": {m.Phone}, "From": {s.senderID},
		"Body": {"[" + s.senderID + "] Your verification code is " + m.Code + ". It expires in 5 minutes."}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.twilio.com/2010-04-01/Accounts/"+s.accountSID+"/Messages.json", strings.NewReader(form.Encode()))
	if err != nil {
		return receipt{}, &Failure{Reason: "invalid_provider_endpoint"}
	}
	req.SetBasicAuth(s.accountSID, s.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var response struct {
		SID  string `json:"sid"`
		Code int    `json:"code"`
	}
	status, err := exchange(s.client, req, &response)
	if err != nil {
		return receipt{}, err
	}
	if status != http.StatusCreated || response.SID == "" {
		return receipt{}, &Failure{Reason: "provider_rejected", HTTPStatus: status, ProviderCode: strconv.Itoa(response.Code)}
	}
	return receipt{Provider: "twilio", MessageID: response.SID}, nil
}
