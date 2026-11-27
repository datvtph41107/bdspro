package phonedelivery

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type zaloSender struct {
	client                     *http.Client
	token, templateID, otpParam string
}

func (s zaloSender) Send(ctx context.Context, m message) (receipt, error) {
	req, err := jsonRequest(ctx, "https://business.openapi.zalo.me/message/template", map[string]any{
		"phone": strings.TrimPrefix(m.Phone, "+"), "template_id": s.templateID,
		"template_data": map[string]string{s.otpParam: m.Code},
		"tracking_id":   strings.ReplaceAll(uuid.NewString(), "-", ""),
	})
	if err != nil {
		return receipt{}, err
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
		return receipt{}, err
	}
	if status != http.StatusOK || response.Error == nil || *response.Error != 0 || response.Data.ID == "" {
		code := ""
		if response.Error != nil {
			code = strconv.Itoa(*response.Error)
		}
		return receipt{}, &Failure{Reason: "provider_rejected", HTTPStatus: status, ProviderCode: code}
	}
	return receipt{Provider: "zalo", MessageID: response.Data.ID}, nil
}
