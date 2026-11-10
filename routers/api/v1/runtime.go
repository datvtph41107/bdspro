package apiv1

import (
	"sync"

	"github.com/datvtph41107/bdspro/modules/ipactive"
	weboauth "github.com/datvtph41107/bdspro/routers/web/oauth"
)

// Runtime owns startup-built capabilities for the API route tree.
// Unlike request state, these are process-scoped and must not change after
// Routes compiles its handlers. Gitea also requires package initialization
// before calling its parameterless route constructors.
type Runtime struct {
	OAuth2Enabled bool
	AuthorizationEnabled bool
	Browser       *weboauth.Handler
	PhoneEnabled  bool
	IPActive      *ipactive.Middleware
	InviteKey     string
}

var (
	runtimeMu sync.Mutex
	runtime   *Runtime
)

// InitRuntime is process startup initialization, not request service lookup.
// Duplicate initialization fails rather than swapping live auth dependencies.
func InitRuntime(capabilities Runtime) {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	if runtime != nil {
		panic("apiv1: runtime already initialized")
	}
	copy := capabilities
	runtime = &copy
}

func getRuntime() Runtime {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	if runtime == nil {
		panic("apiv1: runtime not initialized")
	}
	return *runtime
}
