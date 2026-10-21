package api

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

// This is the checked-in API contract. It is embedded into the same compiled
// binary as Gin; the docs endpoint cannot accidentally serve stale disk data.
//
//go:embed openapi.json
var openAPISpec []byte

const swaggerUIHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <meta name="referrer" content="no-referrer">
  <title>BDSPro — Local API Reference</title>
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.17.14/swagger-ui.min.css">
  <style>html,body{margin:0;background:#fafafa} .topbar{display:none}</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.17.14/swagger-ui-bundle.min.js"></script>
  <script>
    window.addEventListener('load', function () {
      window.ui = SwaggerUIBundle({
        url: '/docs/openapi.json',
        dom_id: '#swagger-ui',
        deepLinking: true,
        displayRequestDuration: true,
        persistAuthorization: false,
        requestInterceptor: function (request) {
          if (['POST', 'PUT', 'PATCH', 'DELETE'].includes((request.method || '').toUpperCase())) {
            request.headers['X-BDSPro-Request'] = '1';
          }
          return request;
        }
      });
    });
  </script>
</body>
</html>`

const docsCSP = "default-src 'none'; connect-src 'self'; " +
	"script-src 'unsafe-inline' https://cdnjs.cloudflare.com; " +
	"style-src 'unsafe-inline' https://cdnjs.cloudflare.com; " +
	"img-src 'self' data:; font-src 'self' data: https://cdnjs.cloudflare.com; " +
	"frame-ancestors 'none'; base-uri 'none'; form-action 'none'"

// Documentation is opt-in by ENVIRONMENT=development. It is deliberately not
// registered at all outside development, even if the browser finds the URL.
// The Swagger UI JS/CSS are pinned CDN assets; the API contract and responses
// are always served from this local binary. Offline UI requires vendored assets
// in a later asset-packaging step.
func (server *Server) serveOpenAPI(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Data(http.StatusOK, "application/json; charset=utf-8", openAPISpec)
}

func (server *Server) serveSwaggerUI(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Header("Content-Security-Policy", docsCSP)
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIHTML))
}
