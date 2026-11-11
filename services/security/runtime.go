package security

import (
	"errors"
	"sync"

	"github.com/datvtph41107/bdspro/modules/conn"
	"github.com/datvtph41107/bdspro/modules/ipactive"
	"github.com/datvtph41107/bdspro/modules/setting"
	redis "github.com/go-redis/redis/v8"
)

var processSecurity struct {
	mu       sync.RWMutex
	client   *redis.Client
	throttle *AuthThrottle
	limiter  *ipactive.Middleware
}

func Init() error {
	client, err := conn.ProcessRedis()
	if err != nil {
		return err
	}
	policies := setting.IPActive.Policies
	if len(policies) == 0 {
		policies = ipactive.DefaultPolicies()
	}
	limiter, err := ipactive.NewRedis(client, policies)
	if err != nil {
		return err
	}
	processSecurity.mu.Lock()
	defer processSecurity.mu.Unlock()
	if processSecurity.client != nil {
		return errors.New("security: already initialized")
	}
	processSecurity.client = client
	processSecurity.throttle = NewAuthThrottle(client)
	processSecurity.limiter = ipactive.NewMiddleware(limiter)
	return nil
}

func ProcessCapabilities() (*redis.Client, *AuthThrottle, *ipactive.Middleware) {
	processSecurity.mu.RLock()
	defer processSecurity.mu.RUnlock()
	if processSecurity.client == nil {
		panic("security: subsystem not initialized")
	}
	return processSecurity.client, processSecurity.throttle, processSecurity.limiter
}
