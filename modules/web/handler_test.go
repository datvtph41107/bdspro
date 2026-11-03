package web

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/datvtph41107/bdspro/modules/reqctx"
	"github.com/datvtph41107/bdspro/modules/web/types"
	apictx "github.com/datvtph41107/bdspro/services/context"
)

// A single registration is analogous to Gitea's package initialization.
func init() {
	RegisterResponseStatusProvider[*apictx.APIContext](apictx.GetAPIContext)
}

func TestTypedContextAndShortCircuit(t *testing.T) {
	order := []string{}
	authenticate := func(ctx *apictx.APIContext) {
		order = append(order, "auth")
		ctx.APIError(http.StatusUnauthorized, "auth.required", "authentication required")
	}
	endpoint := func(ctx *apictx.APIContext) {
		order = append(order, "endpoint")
		ctx.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}
	chain := adaptHandlers(nil, []any{reqctx.RequestContextHandler(), apictx.APIContexter()}, []any{authenticate, endpoint})
	w := httptest.NewRecorder()
	chain(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
	if !reflect.DeepEqual(order, []string{"auth"}) {
		t.Fatalf("unexpected execution %v", order)
	}
}

func TestTypedMiddlewareContinuesWithoutResponse(t *testing.T) {
	order := []string{}
	middleware := func(ctx *apictx.APIContext) {
		order = append(order, "middleware")
		ctx.Doer = &apictx.Principal{AccountID: 8}
	}
	endpoint := func(ctx *apictx.APIContext) {
		order = append(order, "endpoint")
		if ctx.Doer == nil || ctx.Doer.AccountID != 8 {
			t.Fatal("principal unavailable")
		}
		ctx.JSON(200, map[string]int{"id": 8})
	}
	chain := adaptHandlers(nil, []any{reqctx.RequestContextHandler(), apictx.APIContexter()}, []any{middleware, endpoint})
	w := httptest.NewRecorder()
	chain(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != 200 || !reflect.DeepEqual(order, []string{"middleware", "endpoint"}) {
		t.Fatalf("status=%d order=%v", w.Code, order)
	}
}

func TestPreMiddlewareExecutionOrder(t *testing.T) {
	var order []string
	appendMark := func(mark string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, mark)
				next.ServeHTTP(w, r)
			})
		}
	}
	chain := adaptHandlers(
		[]any{types.PreMiddlewareProvider(appendMark("pre-group")), appendMark("group")},
		[]any{types.PreMiddlewareProvider(appendMark("pre-after")), appendMark("after")},
		[]any{func(w http.ResponseWriter, r *http.Request) { order = append(order, "endpoint"); w.WriteHeader(204) }},
	)
	chain(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	want := []string{"pre-after", "pre-group", "after", "group", "endpoint"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("got %v, want %v", order, want)
	}
}

func TestInvalidSignaturesRejectedAtRegistration(t *testing.T) {
	for _, bad := range []any{
		func(int) {}, func() {}, func(*apictx.APIContext) error { return nil },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("expected panic for %T", bad)
				}
			}()
			_ = toHandlerProvider(bad)
		}()
	}
}

func BenchmarkTypedHandler(b *testing.B) {
	endpoint := func(ctx *apictx.APIContext) { ctx.Resp.WriteHeader(http.StatusNoContent) }
	chain := adaptHandlers(nil, []any{reqctx.RequestContextHandler(), apictx.APIContexter()}, []any{endpoint})
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain(httptest.NewRecorder(), r)
	}
}
