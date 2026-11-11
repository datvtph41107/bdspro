package routers

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	authcrypto "github.com/datvtph41107/bdspro/modules/auth"
	"github.com/datvtph41107/bdspro/modules/web"
	appctx "github.com/datvtph41107/bdspro/services/context"
)

// This is an exploratory, within-run latency comparison. CI runners are
// shared hosts; p95/p99 are reported, never treated as production SLOs.
func TestG12SessionAuthorityRealPostgresReadCost(t *testing.T) {
	_, pool, server, _ := authAcceptanceServer(t)
	_, login := g12RegisterAndLogin(t, server)
	access := login["access_token"].(string)
	const samples = 80
	r := web.NewRouter()
	r.AfterRouting(appctx.APIContexter())
	r.Get("/jwt-only", appctx.Authenticate(), func(c *appctx.APIContext) {
		c.Status(http.StatusNoContent)
	})
	r.Get("/live", appctx.Authenticate(), appctx.RequireSession(),
		func(c *appctx.APIContext) {
			c.Status(http.StatusNoContent)
		})
	// Request capability ownership matches Server.Handler. The two routes
	// only differ in whether live session verification reads PostgreSQL.
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := authcrypto.WithMaker(req.Context(), server.tokenMaker)
		r.ServeHTTP(w, req.WithContext(ctx))
	})
	type measured struct {
		durations []time.Duration
		acquires  int64
	}
	run := func(path string) measured {
		out := measured{durations: make([]time.Duration, 0, samples)}
		before := pool.Stat().AcquireCount()
		for i := 0; i < samples; i++ {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Authorization", "Bearer "+access)
			w := httptest.NewRecorder()
			started := time.Now()
			handler.ServeHTTP(w, req)
			out.durations = append(out.durations, time.Since(started))
			if w.Code != http.StatusNoContent {
				t.Fatalf("%s: status=%d body=%s", path, w.Code, w.Body.String())
			}
		}
		out.acquires = pool.Stat().AcquireCount() - before
		sort.Slice(out.durations, func(i, j int) bool {
			return out.durations[i] < out.durations[j]
		})
		return out
	}
	stateless := run("/jwt-only")
	live := run("/live")
	if stateless.acquires != 0 || live.acquires != samples {
		t.Fatalf("unexpected DB acquisition count: jwt-only=%d live=%d, want 0/%d",
			stateless.acquires, live.acquires, samples)
	}
	t.Logf("80 sequential in-process requests (CI runner, not SLO): JWT-only p50=%s p95=%s p99=%s; live session p50=%s p95=%s p99=%s; additional pool acquires/request=1",
		stateless.durations[samples/2], stateless.durations[(samples*95+99)/100-1], stateless.durations[(samples*99+99)/100-1],
		live.durations[samples/2], live.durations[(samples*95+99)/100-1], live.durations[(samples*99+99)/100-1])
}
