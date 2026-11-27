package phonedelivery

import (
	"context"
	"net/http"
)

type telegramSender struct {
	client *http.Client
	token  string
}

func (s telegramSender) Send(ctx context.Context, m message) (receipt, error) {
	req, err := jsonRequest(ctx, "https://gatewayapi.telegram.org/sendVerificationMessage", map[string]any{
		"phone_number": m.Phone, "code": m.Code, "ttl": m.TTLMinutes * 60,
	})
	if err != nil {
		return receipt{}, err
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
		return receipt{}, err
	}
	if status != http.StatusOK || !response.OK || response.Result.ID == "" {
		return receipt{}, &Failure{Reason: "provider_rejected", HTTPStatus: status}
	}
	return receipt{Provider: "telegram_gateway", MessageID: response.Result.ID}, nil
}
