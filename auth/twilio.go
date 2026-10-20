package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TwilioSMS sends OTPs using the fixed official HTTPS Messages endpoint.
// The configured credentials never appear in diagnostics or application logs.
type TwilioSMS struct {
	AccountSID, AuthToken, From string
	Client *http.Client
}

func NewTwilioSMS(accountSID, authToken, from string) SMSSender {
	if accountSID == "" { return nil }
	return TwilioSMS{
		AccountSID:accountSID,AuthToken:authToken,From:from,
		Client:&http.Client{Timeout:10*time.Second},
	}
}

func (s TwilioSMS) SendOTP(ctx context.Context, phone, code string) error {
	values:=url.Values{
		"To":{phone},
		"From":{s.From},
		"Body":{"BDSPro verification code: "+code+". Expires in 5 minutes. Do not share it."},
	}
	endpoint:="https://api.twilio.com/2010-04-01/Accounts/"+s.AccountSID+"/Messages.json"
	req,err:=http.NewRequestWithContext(ctx,http.MethodPost,endpoint,strings.NewReader(values.Encode()))
	if err!=nil{return errors.New("SMS request unavailable")}
	req.SetBasicAuth(s.AccountSID,s.AuthToken)
	req.Header.Set("Content-Type","application/x-www-form-urlencoded")
	client:=s.Client
	if client==nil {client=&http.Client{Timeout:10*time.Second}}
	resp,err:=client.Do(req)
	if err!=nil{return errors.New("SMS provider unavailable")}
	defer resp.Body.Close()
	_,_=io.Copy(io.Discard,io.LimitReader(resp.Body,1024))
	if resp.StatusCode<200 || resp.StatusCode>=300 {
		return fmt.Errorf("SMS provider rejected request (status %d)",resp.StatusCode)
	}
	return nil
}
