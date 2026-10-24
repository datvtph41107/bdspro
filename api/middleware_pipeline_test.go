package api

import (
    "bytes"
    "net"
    "net/http"
    "net/http/httptest"
    "strconv"
    "strings"
    "testing"

    mockdb "github.com/datvtph41107/bdspro/db/mock"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "github.com/stretchr/testify/require"
    "go.uber.org/mock/gomock"
)

func TestPreLoginBlocksMalformedInputBeforeController(t *testing.T) {
    server := newTestServer(t, nil)
    response := httptest.NewRecorder()
    request := httptest.NewRequest(http.MethodPost, "/users/login",
        strings.NewReader(`{"email":"not-an-email","password":"anything"}`))
    request.Header.Set("Content-Type", "application/json")
    server.Handler().ServeHTTP(response, request)
    require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
}

func TestSufLoginOnlyRecordsResult(t *testing.T) {
    var output bytes.Buffer
    original := log.Logger
    log.Logger = zerolog.New(&output)
    t.Cleanup(func() { log.Logger = original })

    ctrl := gomock.NewController(t)
    store := mockdb.NewMockStore(ctrl)
    store.EXPECT().GetAccountByEmail(gomock.Any(), "hello@example.test").
        Return(db.Account{}, pgx.ErrNoRows)

    server := newTestServer(t, store)
    response := httptest.NewRecorder()
    request := httptest.NewRequest(http.MethodPost, "/users/login",
        strings.NewReader(`{"email":"hello@example.test","password":"private-password-123"}`))
    request.Header.Set("Content-Type","application/json")
    server.Handler().ServeHTTP(response,request)
    require.Equal(t,http.StatusUnauthorized,response.Code)
    require.Contains(t,output.String(),`"message":"auth.login_completed"`)
    require.Contains(t,output.String(),`"outcome":"rejected"`)
    require.NotContains(t,output.String(),"private-password-123")
    require.NotContains(t,output.String(),"hello@example.test")
}

func TestSecurityMonitorDetectsRepeatedDenials(t *testing.T) {
    client := redisForAuthRateTest(t)
    var output bytes.Buffer
    original := log.Logger
    log.Logger = zerolog.New(&output)
    t.Cleanup(func() { log.Logger = original })

    server:=newTestServer(t,nil)
    require.NoError(t,server.EnableAuthRateLimiting(client))

    // Random IPv6 per test prevents interaction with other Redis counters.
    id := uuid.New()
    ip := net.IP(id[:]).String()
    for i:=0; i<int(securityDetectionThreshold)+1; i++ {
        request:=httptest.NewRequest(http.MethodGet,
            "/users/me?private_query=secret-never-log-"+strconv.Itoa(i),nil)
        request.RemoteAddr=net.JoinHostPort(ip,"24000")
        request.Header.Set("Authorization","Bearer private-token-never-log")
        response:=httptest.NewRecorder()
        server.Handler().ServeHTTP(response,request)
        require.Equal(t,http.StatusUnauthorized,response.Code)
    }

    require.Equal(t,1,strings.Count(output.String(),`"message":"security.suspicious_activity"`))
    require.Equal(t,int(securityDetectionThreshold)+1,
        strings.Count(output.String(),`"message":"security.request_denied"`))
    require.Contains(t,output.String(),`"reason":"repeated_denials"`)
    require.NotContains(t,output.String(),"private-token-never-log")
    require.NotContains(t,output.String(),"secret-never-log")
    require.NotContains(t,output.String(),ip)
}

func TestSecurityMonitorDoesNotLogUntrustedHeader(t *testing.T) {
    var output bytes.Buffer
    original := log.Logger
    log.Logger = zerolog.New(&output)
    t.Cleanup(func(){ log.Logger=original })

    server := newTestServer(t,nil)
    req := httptest.NewRequest(http.MethodGet,"/users/me",nil)
    req.RemoteAddr="192.0.2.60:1234"
    req.Header.Set("X-Forwarded-For","not-a-real-ip-secret")
    w := httptest.NewRecorder()
    server.Handler().ServeHTTP(w,req)
    require.Equal(t,http.StatusUnauthorized,w.Code)
    require.Contains(t,output.String(),`"message":"security.request_denied"`)
    require.NotContains(t,output.String(),"not-a-real-ip-secret")
}

