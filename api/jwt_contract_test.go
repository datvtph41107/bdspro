package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/stretchr/testify/require"
)

func TestJWTBearerLoginRenewAndLogoutBoundary(t *testing.T) {
	jwtIssuer, err := auth.NewAccessTokens(strings.Repeat("Z",32))
	require.NoError(t,err)
	stub := &authStub{loginResult:auth.LoginResult{
		AccountID: 17,Token:"opaque-session",ExpiresAt:time.Now().Add(24*time.Hour)},accountID:17}
	server:=NewServer(nil,stub,nil,nil,true,"")
	server.EnableAccessTokens(jwtIssuer)

	loginReq:=httptest.NewRequest(http.MethodPost,"/auth/login",strings.NewReader(`{"email":"user@example.test","password":"ValidPassword123!"}`))
	loginReq.Header.Set("Content-Type","application/json")
	loginReq.Header.Set("X-BDSPro-Request","1")
	w:=httptest.NewRecorder()
	server.Handler().ServeHTTP(w,loginReq)
	require.Equal(t,200,w.Code)
	var body struct{
		AccountID int64 `json:"account_id"`
		AccessToken string `json:"access_token"`
		TokenType string `json:"token_type"`
	}
	require.NoError(t,json.Unmarshal(w.Body.Bytes(),&body))
	require.EqualValues(t,17,body.AccountID)
	require.Equal(t,"Bearer",body.TokenType)
	require.NotEmpty(t,body.AccessToken)
	require.NotContains(t,body.AccessToken,"opaque-session")
	require.Len(t,w.Result().Cookies(),1)

	req:=httptest.NewRequest(http.MethodGet,"/auth/me",nil)
	req.Header.Set("Authorization","Bearer "+body.AccessToken)
	w=httptest.NewRecorder()
	server.Handler().ServeHTTP(w,req)
	require.Equal(t,200,w.Code)
	require.Len(t,stub.accountHash,64)
	require.Empty(t,stub.accountToken)

	stub.accountErr=auth.ErrUnauthorized
	w=httptest.NewRecorder()
	server.Handler().ServeHTTP(w,req)
	require.Equal(t,401,w.Code,"revoked DB session must invalidate signed JWT")
	stub.accountErr=nil

	renew:=httptest.NewRequest(http.MethodPost,"/auth/token/renew",nil)
	renew.AddCookie(&http.Cookie{Name:"bdspro_session",Value:"opaque-session"})
	renew.Header.Set("X-BDSPro-Request","1")
	w=httptest.NewRecorder()
	server.Handler().ServeHTTP(w,renew)
	require.Equal(t,200,w.Code)
	require.Contains(t,w.Body.String(),`"access_token":`)

	renew.Header.Set("X-BDSPro-Request","")
	w=httptest.NewRecorder()
	server.Handler().ServeHTTP(w,renew)
	require.Equal(t,403,w.Code)
}

func TestJWTInvalidBearerNeverDowngradesToCookie(t *testing.T) {
	issuer,err:=auth.NewAccessTokens(strings.Repeat("Z",32))
	require.NoError(t,err)
	stub:=&authStub{accountID:17}
	server:=NewServer(nil,stub,nil,nil,true,"")
	server.EnableAccessTokens(issuer)
	req:=httptest.NewRequest(http.MethodGet,"/auth/me",nil)
	req.AddCookie(&http.Cookie{Name:"bdspro_session",Value:"valid-cookie"})
	req.Header.Set("Authorization","Bearer invalid")
	w:=httptest.NewRecorder()
	server.Handler().ServeHTTP(w,req)
	require.Equal(t,401,w.Code)
	require.Zero(t,stub.accountCalls)
}
