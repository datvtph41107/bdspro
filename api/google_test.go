package api

import (
 "bytes"
 "net/http"
 "net/http/httptest"
 "testing"

 mockdb "github.com/datvtph41107/bdspro/db/mock"
 db "github.com/datvtph41107/bdspro/db/sqlc"
 "github.com/datvtph41107/bdspro/util"
 "github.com/go-redis/redis/v8"
 "github.com/stretchr/testify/require"
 "go.uber.org/mock/gomock"
)

func TestGoogleRoutesDisabledWithoutCredentials(t *testing.T) {
 server:=newTestServer(t,nil)
 require.NoError(t,server.EnableGoogleOAuth(nil))
 w:=httptest.NewRecorder()
 server.router.ServeHTTP(w,httptest.NewRequest(http.MethodGet,"/users/google/login/start",nil))
 require.Equal(t,http.StatusNotFound,w.Code)
}

func TestGoogleLinkRequiresAccessTokenAndCorrectPassword(t *testing.T) {
 ctrl:=gomock.NewController(t)
 store:=mockdb.NewMockStore(ctrl)
 server:=newTestServer(t,store)
 server.config.GoogleClientID="test-client"
 server.config.GoogleClientSecret="test-secret"
 server.config.GoogleRedirectURI="http://localhost:8080/users/google/callback"
 client:=redis.NewClient(&redis.Options{Addr:"127.0.0.1:1"})
 defer client.Close()
 require.NoError(t,server.EnableGoogleOAuth(client))
 w:=httptest.NewRecorder()
 req:=httptest.NewRequest(http.MethodPost,"/users/me/google/start",bytes.NewBufferString(`{"password":"incorrect"}`))
 req.Header.Set("Content-Type","application/json")
 server.router.ServeHTTP(w,req)
 require.Equal(t,http.StatusUnauthorized,w.Code)
 hash,err:=util.HashPassword("correct-password")
 require.NoError(t,err)
 store.EXPECT().GetAccountByID(gomock.Any(),int64(10)).Return(db.Account{ID:10,PasswordHash:hash},nil)
 w=httptest.NewRecorder()
 server.router.ServeHTTP(w,authRequest(t,server,http.MethodPost,"/users/me/google/start",`{"password":"incorrect"}`))
 require.Equal(t,http.StatusUnauthorized,w.Code)
 w=httptest.NewRecorder()
 server.router.ServeHTTP(w,httptest.NewRequest(http.MethodGet,"/users/google/callback",nil))
 require.Equal(t,http.StatusBadRequest,w.Code)
}
