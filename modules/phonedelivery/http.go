package phonedelivery

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

const maxResponseBytes = 16 << 10

func jsonRequest(ctx context.Context, endpoint string, value any) (*http.Request, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, &Failure{Reason: "invalid_delivery_message"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, &Failure{Reason: "invalid_provider_endpoint"}
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func exchange(client *http.Client, req *http.Request, value any) (int, error) {
	response, err := client.Do(req)
	if err != nil {
		// URL errors may contain credentials or parameters: do not expose them.
		return 0, &Failure{Reason: "provider_transport_error"}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		return response.StatusCode, &Failure{Reason: "invalid_provider_response", HTTPStatus: response.StatusCode}
	}
	if err := json.Unmarshal(body, value); err != nil {
		return response.StatusCode, &Failure{Reason: "invalid_provider_response", HTTPStatus: response.StatusCode}
	}
	return response.StatusCode, nil
}
