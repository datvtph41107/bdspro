package common

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)
func TestRequestIDHeaderAndContext(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r:=gin.New()
    r.Use(RequestID())
    r.GET("/check",func(c *gin.Context){
        if c.Writer.Header().Get("X-Request-ID")=="" {t.Error("no request id")}
        c.Status(http.StatusNoContent)
    })
    out:=httptest.NewRecorder()
    r.ServeHTTP(out,httptest.NewRequest(http.MethodGet,"/check",nil))
    if out.Code!=http.StatusNoContent||out.Header().Get("X-Request-ID")=="" {
        t.Fatalf("bad middleware response: %d %v",out.Code,out.Header())
    }
}
