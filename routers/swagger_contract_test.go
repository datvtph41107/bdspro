package routers

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	apiv1 "github.com/datvtph41107/bdspro/routers/api/v1"
	"github.com/datvtph41107/bdspro/routers/api/v1/oauth"
	"github.com/datvtph41107/bdspro/routers/api/v1/org"
	"github.com/datvtph41107/bdspro/routers/api/v1/user"
	weboauth "github.com/datvtph41107/bdspro/routers/web/oauth"
	"github.com/datvtph41107/bdspro/templates"
	"github.com/gin-gonic/gin"
)

// The OpenAPI contract must cover every registered Gin route, including
// optionally enabled provider endpoints; it must not invent unimplemented paths.
func TestOpenAPICoverage(t *testing.T) {
	raw, err := templates.Swagger.ReadFile("swagger/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		OpenAPI string                                `json:"openapi"`
		Paths   map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.OpenAPI != "3.0.3" {
		t.Fatalf("unexpected version: %s", spec.OpenAPI)
	}
	gin.SetMode(gin.TestMode)
	s := &Server{
		authHandler:          user.NewAuthHandler(nil, nil, nil),
		organizationHandler:  org.NewHandler(nil),
		oauth2Handler:        user.NewOAuth2Handler(nil, nil, false),
		authorizationHandler: oauth.NewHandler(nil, nil),
		browserHandler: &weboauth.Handler{},
		phoneEnabled:         true,
	}
	// Registration only; no handler is executed and no external services are used.
	root := gin.New()
	apiv1.Routes(root, apiv1.Dependencies{
		Auth:          s.authHandler,
		Organization:  s.organizationHandler,
		OAuth2:        s.oauth2Handler,
		Authorization: s.authorizationHandler,
		Browser: s.browserHandler,
		PhoneEnabled:  s.phoneEnabled,
		TokenMaker:    s.tokenMaker,
		Store:         s.store,
	})
	normalize := regexp.MustCompile(`:[A-Za-z_][A-Za-z0-9_]*`)
	actual := map[string]bool{}
	for _, route := range root.Routes() {
		path := normalize.ReplaceAllStringFunc(route.Path, func(p string) string { return "{" + p[1:] + "}" })
		key := strings.ToLower(route.Method) + " " + path
		actual[key] = true
		if _, ok := spec.Paths[path][strings.ToLower(route.Method)]; !ok {
			t.Errorf("undocumented HTTP operation: %s", key)
		}
	}
	for path, methods := range spec.Paths {
		for method := range methods {
			if !actual[method+" "+path] {
				t.Errorf("OpenAPI operation has no runtime route: %s %s", method, path)
			}
		}
	}
}
