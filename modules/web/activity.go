package web

import (
	"context"
	"net/http"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
)

// HandlerActivity is immutable after route compilation.
type HandlerActivity struct {
	Name   string
	Source string
}

type activityKey struct{}

type RequestActivity struct {
	last atomic.Pointer[HandlerActivity]
}

func WithRequestActivity(ctx context.Context) (context.Context, *RequestActivity) {
	activity := &RequestActivity{}
	return context.WithValue(ctx, activityKey{}, activity), activity
}

// LastHandler is the most recently entered instrumented action, not a stack trace.
func (a *RequestActivity) LastHandler() *HandlerActivity {
	return a.last.Load()
}

func recordHandler(ctx context.Context, handler *HandlerActivity) {
	if handler == nil {
		return
	}
	if activity, ok := ctx.Value(activityKey{}).(*RequestActivity); ok {
		activity.last.Store(handler)
	}
}

func handlerActivity(fn reflect.Value) *HandlerActivity {
	f := runtime.FuncForPC(fn.Pointer())
	if f == nil {
		return nil
	}
	const prefix = "github.com/datvtph41107/bdspro/"
	name, ownPackage := strings.CutPrefix(f.Name(), prefix)
	if !ownPackage {
		// No absolute build paths or false application source attribution.
		return &HandlerActivity{Name: f.Name()}
	}
	start := strings.LastIndexByte(name, '/') + 1
	dot := strings.IndexByte(name[start:], '.')
	if dot < 0 {
		return &HandlerActivity{Name: name}
	}
	file, line := f.FileLine(f.Entry())
	source := path.Join(name[:start+dot], path.Base(strings.ReplaceAll(file, "\\", "/")))
	return &HandlerActivity{
		Name: name[start:],
		Source: source + ":" + strconv.Itoa(line),
	}
}

func observeProvider(provider middlewareProvider, info *HandlerActivity) middlewareProvider {
	return func(next http.Handler) http.Handler {
		h := provider(next)
		if h == nil {
			return nil
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recordHandler(r.Context(), info)
			h.ServeHTTP(w, r)
		})
	}
}
