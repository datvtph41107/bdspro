package api

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/util"
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
		{name: "Allowed", origin: "http://localhost:3000", method: http.MethodGet, want: http.StatusOK, allow: "http://localhost:3000"},
		{name: "Denied", origin: "https://other.example", method: http.MethodGet, want: http.StatusOK},
		{name: "Preflight", origin: "http://localhost:3000", method: http.MethodOptions, want: http.StatusNoContent, allow: "http://localhost:3000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/health", nil)
			req.Header.Set("Origin", tc.origin)
			if tc.method == http.MethodOptions {
				req.Header.Set("Access-Control-Request-Method", "GET")
			}
			handler.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code)
			require.Equal(t, tc.allow, w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

func TestGracefulShutdown(t *testing.T) {
	server, err := NewServer(util.Config{TokenSymmetricKey: "12345678901234567890123456789012"}, nil, testTaskDistributor{})
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
	err = server.StartWithReady(context.Background(), listener.Addr().String(), func(net.Addr) {
		ready = true
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "listen HTTP")
	require.False(t, ready, "must not announce readiness when bind fails")
}

func TestHTTPListeningAndShutdown(t *testing.T) {
	server := newTestServer(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addresses := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		done <- server.StartWithReady(ctx, "127.0.0.1:0", func(addr net.Addr) {
			addresses <- addr.String()
		})
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
