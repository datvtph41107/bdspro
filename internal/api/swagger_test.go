package api

import (
 "encoding/json"
 "io/fs"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 "github.com/datvtph41107/bdspro/doc/statik"
 "github.com/datvtph41107/bdspro/internal/api/handler"
 "github.com/datvtph41107/bdspro/internal/config"
 "github.com/gin-gonic/gin"
)

func TestOpenAPICoversAllRegisteredRoutes(t *testing.T) {
 raw,err:=fs.ReadFile(statik.Files(),"openapi.json")
 if err!=nil{t.Fatal(err)}
 var doc struct{ Paths map[string]map[string]json.RawMessage `json:"paths"` }
 if err:=json.Unmarshal(raw,&doc);err!=nil{t.Fatal(err)}
 gin.SetMode(gin.TestMode)
 srv:=&Server{authHandler:handler.NewAuthHandler(nil,nil),organizationHandler:handler.NewOrganizationHandler(nil),googleHandler:handler.NewGoogleHandler(nil,config.Config{},nil,nil,false),phoneHandler:handler.NewPhoneHandler(nil,config.Config{},nil,nil)}
 srv.setupRouter()
 operations:=0
 for _,rt:=range srv.router.Routes(){
  if strings.HasPrefix(rt.Path,"/swagger/"){continue}
  path:=strings.ReplaceAll(rt.Path,":id","{id}")
  if len(doc.Paths[path][strings.ToLower(rt.Method)])==0 {t.Errorf("missing OpenAPI operation: %s %s",rt.Method,path)}
  operations++
 }
 if operations!=19{t.Fatalf("expected 19 registered operations, got %d",operations)}
}

func TestSwaggerServedInDevelopmentOnly(t *testing.T) {
 gin.SetMode(gin.TestMode)
 r:=gin.New()
 (&Server{config:config.Config{Environment:"development"}}).registerSwagger(r)
 w:=httptest.NewRecorder()
 r.ServeHTTP(w,httptest.NewRequest(http.MethodGet,"/swagger/openapi.json",nil))
 if w.Code!=http.StatusOK{t.Fatalf("swagger status=%d",w.Code)}
 r2:=gin.New()
 (&Server{config:config.Config{Environment:"production"}}).registerSwagger(r2)
 w2:=httptest.NewRecorder()
 r2.ServeHTTP(w2,httptest.NewRequest(http.MethodGet,"/swagger/openapi.json",nil))
 if w2.Code!=http.StatusNotFound {t.Fatalf("swagger exposed in production: %d",w2.Code)}
}
