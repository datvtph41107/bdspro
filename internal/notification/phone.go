package notification

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// PhoneCodeSender delivers a code. Authentication and verification remain
// inside BDSPro; a successful provider response does not verify a phone.
type PhoneCodeSender interface {
	SendCode(ctx context.Context, phone, code string) error
}

// Local is strictly a development transport. The disposable dev database
// already holds the code for manual verification.
type Local struct{}

func (Local) SendCode(context.Context, string, string) error { return nil }

// SMSGateway uses the BDSPro gateway contract: POST {phone,message} with a
// Bearer token. Individual SMS vendors must be adapted behind this contract.
type SMSGateway struct {
	URL   string
	Token string
	Client *http.Client
}

func (s SMSGateway) SendCode(ctx context.Context, phone, code string) error {
	if s.URL == "" || s.Token == "" || s.Client == nil {
		return errors.New("SMS gateway is not configured")
	}
	data, err := json.Marshal(struct {
		Phone   string `json:"phone"`
		Message string `json:"message"`
	}{Phone: phone, Message: "Your verification code is " + code})
	if err != nil { return err }
	return postJSON(ctx, s.Client, s.URL, "Authorization", "Bearer "+s.Token, data, false)
}

// ZaloZBS sends an approved phone-number template using the official ZBS API.
// Access tokens and approved template data names are managed by the operator.
type ZaloZBS struct {
	AccessToken string
	TemplateID  string
	CodeField   string
	Client      *http.Client
}

func (s ZaloZBS) SendCode(ctx context.Context, phone, code string) error {
	if s.AccessToken == "" || s.TemplateID == "" || s.CodeField == "" || s.Client == nil {
		return errors.New("Zalo ZBS delivery is not configured")
	}
	trackingBytes := make([]byte, 16)
	if _, err := rand.Read(trackingBytes); err != nil { return err }
	data, err := json.Marshal(struct {
		TrackingID string            `json:"tracking_id"`
		Phone        string            `json:"phone"`
		TemplateID   string            `json:"template_id"`
		TemplateData map[string]string `json:"template_data"`
	}{
		TrackingID: hex.EncodeToString(trackingBytes),
		Phone: strings.TrimPrefix(phone, "+"),
		TemplateID: s.TemplateID,
		TemplateData: map[string]string{s.CodeField: code},
	})
	if err != nil { return err }
	return postJSON(ctx, s.Client, "https://business.openapi.zalo.me/message/template", "access_token", s.AccessToken, data, true)
}

func postJSON(ctx context.Context, client *http.Client, endpoint, authHeader, authValue string, data []byte, zalo bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil { return err }
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(authHeader, authValue)
	resp, err := client.Do(req)
	if err != nil { return fmt.Errorf("phone delivery failed: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("phone provider rejected request (HTTP %d)", resp.StatusCode)
	}
	if zalo {
		var result struct {
			Error *int `json:"error"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result); err != nil {
			return fmt.Errorf("invalid Zalo delivery response: %w", err)
		}
		if result.Error == nil || *result.Error != 0 {
			return errors.New("Zalo provider did not accept the message")
		}
	}
	return nil
}
