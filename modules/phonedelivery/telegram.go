package phonedelivery

import (
	"context"
	"net/http"
)

func NewTelegramGateway(client *http.Client, token string) *TelegramGateway {
	return &TelegramGateway{client: client, token: token}
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
