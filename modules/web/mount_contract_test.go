package web

import (
	"net/http"
	"testing"
)

func TestMountedBeforeRoutingFailsAtCompilation(t *testing.T) {
	root := NewRouter()
	child := NewRouter()
	child.BeforeRouting(func(next http.Handler) http.Handler { return next })
	child.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	root.Mount("/api", child)
	defer func() {
		if recover() == nil {
			t.Fatal("mount must reject unsupported child BeforeRouting instead of dropping it")
		}
	}()
	root.Compile()
}
