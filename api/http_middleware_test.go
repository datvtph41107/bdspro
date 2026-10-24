package api

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

func TestCORS(t *testing.T) {
	server := newTestServer(t, nil)
	server.config.AllowedOrigins = []string{"http://localhost:3000"}
	handler := server.Handler()

	for _, tc := range []struct {
		name   string
		origin string
		method string
		want   int
		allow  string
	}{
		{
			name:   "Allowed",
			origin: "http://localhost:3000",
			method: http.MethodGet,
			want:   http.StatusOK,
			allow:  "http://localhost:3000",
		},
		{
			name:   "Denied",
			origin: "https://other.example",
			method: http.MethodGet,
			want:   http.StatusOK,
		},
		{
			name:   "Preflight",
			origin: "http://localhost:3000",
			method: http.MethodOptions,
			want:   http.StatusNoContent,
			allow:  "http://localhost:3000",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/health", nil)

			req.Header.Set("Origin", tc.origin)

			if tc.method == http.MethodOptions {
				req.Header.Set(
					"Access-Control-Request-Method",
					"GET",
				)
			}

			handler.ServeHTTP(w, req)

			require.Equal(t, tc.want, w.Code)
			require.Equal(
				t,
				tc.allow,
				w.Header().Get("Access-Control-Allow-Origin"),
			)

			if tc.name == "Allowed" {
				require.True(
					t,
					strings.EqualFold(
						w.Header().Get("Access-Control-Expose-Headers"),
						"X-Request-ID",
					),
				)
				require.NotEmpty(
					t,
					w.Header().Get("X-Request-ID"),
				)
			}
		})
	}
}

func TestGracefulShutdown(t *testing.T) {
	server, err := NewServer(
		util.Config{
			TokenSymmetricKey: "12345678901234567890123456789012",
		},
		nil,
		testTaskDistributor{},
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, server.Start(ctx, "127.0.0.1:0"))
}

func TestStartPortInUse(t *testing.T) {
	server := newTestServer(t, nil)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	ready := false

	err = server.StartWithReady(
		context.Background(),
		listener.Addr().String(),
		func(net.Addr) {
			ready = true
		},
	)

	require.Error(t, err)
	require.ErrorContains(t, err, "listen HTTP")
	require.False(
		t,
		ready,
		"must not announce readiness when bind fails",
	)
}

func TestHTTPListeningAndShutdown(t *testing.T) {
	server := newTestServer(t, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addresses := make(chan string, 1)
	done := make(chan error, 1)

	go func() {
		done <- server.StartWithReady(
			ctx,
			"127.0.0.1:0",
			func(addr net.Addr) {
				addresses <- addr.String()
			},
		)
	}()

	var address string

	select {
	case address = <-addresses:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not bind a TCP port")
	}

	response, err := http.Get("http://" + address + "/health")
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, response.Body.Close())

	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP server did not shut down")
	}
}

func TestHTTPLoggerDoesNotExposeQuerySecrets(t *testing.T) {
	var output bytes.Buffer

	original := log.Logger
	log.Logger = zerolog.New(&output)

	t.Cleanup(func() {
		log.Logger = original
	})

	server := newTestServer(t, nil)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/health?secret_code=private-value",
		nil,
	)

	server.Handler().ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, output.String(), "http.request")
	require.Contains(
		t,
		output.String(),
		`"path":"/health"`,
	)
	require.NotContains(t, output.String(), "secret_code")
	require.NotContains(t, output.String(), "private-value")
}

// Recovery must not expose query parameters, panic values,
// or Authorization credentials even in Gin DebugMode.
func TestHTTPRecoveryDoesNotExposeSecrets(t *testing.T) {
	const (
		querySecret  = "query-secret-test-only"
		panicSecret  = "panic-secret-test-only"
		headerSecret = "header-secret-test-only"
	)

	var output bytes.Buffer

	originalLogger := log.Logger
	log.Logger = zerolog.New(&output)

	t.Cleanup(func() {
		log.Logger = originalLogger
	})

	originalGinWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &output

	t.Cleanup(func() {
		gin.DefaultErrorWriter = originalGinWriter
	})

	server := newTestServer(t, nil)

	server.router.GET("/test/panic", func(ctx *gin.Context) {
		panic(panicSecret)
	})

	originalMode := gin.Mode()
	gin.SetMode(gin.DebugMode)

	t.Cleanup(func() {
		gin.SetMode(originalMode)
	})

	response := httptest.NewRecorder()

	request := httptest.NewRequest(
		http.MethodGet,
		"/test/panic?secret_code="+querySecret,
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer "+headerSecret,
	)

	server.Handler().ServeHTTP(response, request)

	require.Equal(
		t,
		http.StatusInternalServerError,
		response.Code,
	)

	requestID := response.Header().Get("X-Request-ID")
	require.NotEmpty(t, requestID)

	require.Equal(
		t,
		2,
		strings.Count(
			output.String(),
			`"request_id":"`+requestID+`"`,
		),
	)

	require.Contains(
		t,
		output.String(),
		`"message":"http.panic"`,
	)

	require.Contains(
		t,
		output.String(),
		`"path":"/test/panic"`,
	)

	require.Contains(
		t,
		output.String(),
		`"panic_type":"string"`,
	)

	require.Contains(
		t,
		output.String(),
		`"status":500`,
	)

	require.NotContains(t, output.String(), querySecret)
	require.NotContains(t, output.String(), panicSecret)
	require.NotContains(t, output.String(), headerSecret)
}

// Each HTTP request receives a server-generated Request ID.
// The same ID is available to handlers through request context.
func TestHTTPRequestID(t *testing.T) {
	var output bytes.Buffer

	original := log.Logger
	log.Logger = zerolog.New(&output)

	t.Cleanup(func() {
		log.Logger = original
	})

	server := newTestServer(t, nil)

	server.router.GET(
		"/test/request-id",
		func(ctx *gin.Context) {
			log.Ctx(ctx.Request.Context()).
				Info().
				Msg("handler.called")

			ctx.Status(http.StatusNoContent)
		},
	)

	// First request.
	requestA := httptest.NewRequest(
		http.MethodGet,
		"/test/request-id",
		nil,
	)

	requestA.Header.Set(
		"X-Request-ID",
		"client-controlled-value",
	)

	responseA := httptest.NewRecorder()
	server.Handler().ServeHTTP(responseA, requestA)

	require.Equal(
		t,
		http.StatusNoContent,
		responseA.Code,
	)

	idA := responseA.Header().Get("X-Request-ID")

	_, err := uuid.Parse(idA)
	require.NoError(t, err)

	require.NotEqual(
		t,
		"client-controlled-value",
		idA,
	)

	require.Contains(
		t,
		output.String(),
		`"request_id":"`+idA+`"`,
	)

	require.Equal(
		t,
		2,
		strings.Count(
			output.String(),
			`"request_id":"`+idA+`"`,
		),
	)

	// Second request.
	requestB := httptest.NewRequest(
		http.MethodGet,
		"/test/request-id",
		nil,
	)

	responseB := httptest.NewRecorder()
	server.Handler().ServeHTTP(responseB, requestB)

	require.Equal(
		t,
		http.StatusNoContent,
		responseB.Code,
	)

	idB := responseB.Header().Get("X-Request-ID")

	_, err = uuid.Parse(idB)
	require.NoError(t, err)

	require.NotEqual(t, idA, idB)

	require.Equal(
		t,
		2,
		strings.Count(
			output.String(),
			`"request_id":"`+idB+`"`,
		),
	)
}

func TestRuntimeRefusesMissingIPActiveLimiter(t *testing.T) {
    server:=newTestServer(t,nil)
    server.config.Environment="development"
    called:=false
    err:=server.StartWithReady(context.Background(),"127.0.0.1:0",func(net.Addr){called=true})
    require.ErrorContains(t,err,"IPActive Redis limiter")
    require.False(t,called)
}
