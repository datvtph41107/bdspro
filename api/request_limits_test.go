package api

import (
    "bytes"
    "context"
    "fmt"
    "net/http"
    "net/http/httptest"
    "os"
    "strings"
    "sync"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
    "github.com/google/uuid"
    "github.com/stretchr/testify/require"
)

func TestRequestBodySizeLimit(t *testing.T) {
    server := newTestServer(t, nil)
    body := `{"email":"` + strings.Repeat("x",int(maxRequestBodyBytes)) + `","password":"password123"}`
    for _,tc := range []struct{name string; chunked bool}{
        {name:"declared content length"},
        {name:"chunked body",chunked:true},
    }{
        t.Run(tc.name,func(t *testing.T){
            request:=httptest.NewRequest(http.MethodPost,"/users",strings.NewReader(body))
            request.Header.Set("Content-Type","application/json")
            if tc.chunked { request.ContentLength = -1 }
            response:=httptest.NewRecorder()
            server.router.ServeHTTP(response,request)
            require.Equal(t,http.StatusRequestEntityTooLarge,response.Code,response.Body.String())
            require.NotEmpty(t,response.Header().Get("X-Request-ID"))
        })
    }
}

func TestUntrustedProxyCannotSpoofClientIP(t *testing.T) {
    server:=newTestServer(t,nil)
    server.router.GET("/test/ip",func(ctx *gin.Context){
        ctx.String(http.StatusOK,ctx.ClientIP())
    })
    request:=httptest.NewRequest(http.MethodGet,"/test/ip",nil)
    request.RemoteAddr="192.0.2.10:3421"
    request.Header.Set("X-Forwarded-For","198.51.100.12")
    request.Header.Set("X-Real-IP","198.51.100.12")
    response:=httptest.NewRecorder()
    server.router.ServeHTTP(response,request)
    require.Equal(t,http.StatusOK,response.Code)
    require.Equal(t,"192.0.2.10",response.Body.String())
}

func TestAuthRateLimitUnavailableRejectsLogin(t *testing.T) {
    client:=redis.NewClient(&redis.Options{
        Addr:"127.0.0.1:1",
        DialTimeout:100*time.Millisecond,
        ReadTimeout:100*time.Millisecond,
        WriteTimeout:100*time.Millisecond,
        MaxRetries:0,
    })
    defer client.Close()
    server:=newTestServer(t,nil)
    require.NoError(t,server.EnableAuthRateLimiting(client))
    response:=httptest.NewRecorder()
    request:=httptest.NewRequest(http.MethodPost,"/users/login",
        bytes.NewBufferString(`{"email":"test@example.test","password":"wrong-password"}`))
    request.Header.Set("Content-Type","application/json")
    server.router.ServeHTTP(response,request)
    require.Equal(t,http.StatusServiceUnavailable,response.Code)
}

func redisForAuthRateTest(t *testing.T) *redis.Client {
    t.Helper()
    addr:=os.Getenv("REDIS_TEST_ADDRESS")
    if addr=="" {t.Skip("Redis integration test requires REDIS_TEST_ADDRESS")}
    client:=redis.NewClient(&redis.Options{Addr:addr})
    require.NoError(t,client.Ping(context.Background()).Err())
    t.Cleanup(func(){require.NoError(t,client.Close())})
    return client
}

func TestAuthRateLimitSharedCountersAndReplay(t *testing.T) {
    client:=redisForAuthRateTest(t)
    server:=newTestServer(t,nil)
    require.NoError(t,server.EnableAuthRateLimiting(client))
    operation:="test-"+uuid.NewString()
    server.router.GET("/test/rate",func(ctx *gin.Context){
        if !server.allowAuthRequest(ctx,operation,"Shared@Example.Test",50,2,time.Minute){return}
        ctx.Status(http.StatusNoContent)
    })
    send:=func(ip string) *httptest.ResponseRecorder {
        req:=httptest.NewRequest(http.MethodGet,"/test/rate",nil)
        req.RemoteAddr=ip+":3456"
        req.Header.Set("X-Forwarded-For",uuid.NewString())
        response:=httptest.NewRecorder()
        server.router.ServeHTTP(response,req)
        return response
    }
    require.Equal(t,http.StatusNoContent,send("192.0.2.20").Code)
    require.Equal(t,http.StatusNoContent,send("192.0.2.21").Code)
    response:=send("192.0.2.22")
    require.Equal(t,http.StatusTooManyRequests,response.Code)
    require.Equal(t,"60",response.Header().Get("Retry-After"))
}

func TestAuthRateLimitAtomicUnderConcurrency(t *testing.T) {
    client:=redisForAuthRateTest(t)
    operation:="concurrency-"+uuid.NewString()
    const attempts=24
    const allowed=5
    var wg sync.WaitGroup
    results:=make(chan int64,attempts)
    errs:=make(chan error,attempts)
    for i:=0;i<attempts;i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            namespace:="bdspro:auth_limit:v1:"+operation+":"
            result,err:=authRateScript.Run(context.Background(),client,[]string{
                namespace+"ip:"+hashedRateKey("192.0.2.40"),
                namespace+"identity:"+hashedRateKey("alice@example.test"),
            },60,allowed,allowed).Int64()
            if err!=nil {errs<-err;return}
            results<-result
        }()
    }
    wg.Wait()
    close(errs)
    close(results)
    for err:=range errs{require.NoError(t,err)}
    success:=0
    for result:=range results{if result==1 {success++}}
    require.Equal(t,allowed,success,fmt.Sprintf("only %d of %d concurrent attempts may pass",allowed,attempts))
}

func TestAuthRateLimiterRequiredWhenEnabled(t *testing.T) {
    server:=newTestServer(t,nil)
    require.Error(t,server.EnableAuthRateLimiting(nil))
}
