package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type openAPIOperation struct {
	Summary    string                     `json:"summary"`
	Responses  map[string]json.RawMessage `json:"responses"`
	Parameters []struct {
		Name     string `json:"name"`
		In       string `json:"in"`
		Required bool   `json:"required"`
	} `json:"parameters"`
	Security    []map[string]json.RawMessage `json:"security"`
	RequestBody *json.RawMessage             `json:"requestBody"`
}

type openAPIContract struct {
	OpenAPI    string                                 `json:"openapi"`
	Paths      map[string]map[string]openAPIOperation `json:"paths"`
	Components struct {
		Schemas         map[string]json.RawMessage `json:"schemas"`
		SecuritySchemes map[string]json.RawMessage `json:"securitySchemes"`
	} `json:"components"`
}

func readOpenAPI(t *testing.T) openAPIContract {
	t.Helper()
	var spec openAPIContract
	require.NoError(t, json.Unmarshal(openAPISpec, &spec))
	require.Equal(t, "3.1.0", spec.OpenAPI)
	require.NotEmpty(t, spec.Components.Schemas)
	require.NotEmpty(t, spec.Components.SecuritySchemes["bearerAuth"])
	require.NotEmpty(t, spec.Components.SecuritySchemes["cookieAuth"])
	require.NotEmpty(t, spec.Components.SecuritySchemes["hostCookieAuth"])
	return spec
}

var ginPathParameter = regexp.MustCompile(`:[a-zA-Z][a-zA-Z0-9]*`)

// A change to Gin route registration must update OpenAPI in the same PR.
// This checks every method/path in BOTH directions, not just a snapshot count.
func TestOpenAPIExactlyMatchesGinApplicationRoutes(t *testing.T) {
	spec := readOpenAPI(t)
	server := NewServer(nil, nil, nil, nil, true, "")
	live := map[string]bool{}
	for _, route := range server.router.Routes() {
		if strings.HasPrefix(route.Path, "/docs") {
			continue
		}
		method := strings.ToLower(route.Method)
		path := ginPathParameter.ReplaceAllStringFunc(route.Path, func(p string) string {
			return "{" + p[1:] + "}"
		})
		key := method + " " + path
		live[key] = true
		operations, ok := spec.Paths[path]
		require.True(t, ok, "live Gin route missing from OpenAPI: %s", key)
		op, ok := operations[method]
		require.True(t, ok, "live Gin method missing from OpenAPI: %s", key)
		require.NotEmpty(t, op.Summary, key)
		var succeeded bool
		for status, data := range op.Responses {
			require.NotEmpty(t, data, key+" "+status)
			if strings.HasPrefix(status, "2") {
				succeeded = true
			}
		}
		require.True(t, succeeded, "no success response in "+key)
		for _, match := range regexp.MustCompile(`\{([a-zA-Z][a-zA-Z0-9]*)\}`).FindAllStringSubmatch(path, -1) {
			var found bool
			for _, p := range op.Parameters {
				if p.Name == match[1] && p.In == "path" && p.Required {
					found = true
				}
			}
			require.True(t, found, "missing required OpenAPI path param %s in %s", match[1], key)
		}
	}
	for path, operations := range spec.Paths {
		for method := range operations {
			require.True(t, live[method+" "+path], "documented route is not implemented by Gin: %s %s", method, path)
		}
	}
	require.Len(t, live, 29)
}

func TestOpenAPIAuthAndSourceContracts(t *testing.T) {
	spec := readOpenAPI(t)
	login := spec.Paths["/auth/login"]["post"]
	require.NotNil(t, login.RequestBody)
	require.Contains(t, login.Responses, "200")
	require.Empty(t, login.Security, "login must not require a pre-existing Session")

	renew := spec.Paths["/auth/token/renew"]["post"]
	require.Nil(t, renew.RequestBody, "JWT renewal does not take a JSON body")
	require.Len(t, renew.Security, 2, "renewal is cookie-only for either deployment cookie name")
	for _, mode := range renew.Security {
		require.NotContains(t, mode, "bearerAuth")
	}

	protected := spec.Paths["/organizations/{id}/sources"]["post"]
	require.NotNil(t, protected.RequestBody)
	require.Contains(t, protected.Responses, "201")
	require.Contains(t, protected.Responses, "200")
	var hasIdempotency bool
	for _, p := range protected.Parameters {
		if p.Name == "Idempotency-Key" {
			hasIdempotency = p.Required
		}
	}
	require.True(t, hasIdempotency)
	require.Len(t, protected.Security, 3)
}

func TestSwaggerIsDevelopmentOnlyAndEmbedsSameSpec(t *testing.T) {
	dev := NewServer(nil, nil, nil, nil, true, "")
	w := httptest.NewRecorder()
	dev.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, string(openAPISpec), w.Body.String())
	require.Contains(t, w.Header().Get("Content-Type"), "application/json")
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))

	w = httptest.NewRecorder()
	dev.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/docs", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "SwaggerUIBundle")
	require.Contains(t, w.Body.String(), "/docs/openapi.json")
	require.Contains(t, w.Body.String(), "X-BDSPro-Request")
	require.Contains(t, w.Header().Get("Content-Type"), "text/html")
	require.NotEmpty(t, w.Header().Get("Content-Security-Policy"))
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))

	production := NewServer(nil, nil, nil, nil, false, "")
	for _, path := range []string{"/docs", "/docs/openapi.json"} {
		w = httptest.NewRecorder()
		production.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNotFound, w.Code, path)
		require.NotContains(t, w.Body.String(), "SwaggerUIBundle")
	}
}
