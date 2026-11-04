package cache

import (
	"fmt"

	"github.com/datvtph41107/bdspro/modules/setting"
	redis "github.com/go-redis/redis/v8"
)

// NewStringCache selects a cache backend, without opening or owning Redis
// connections. Redis clients are provided by modules/connection.
func NewStringCache(cfg setting.CacheSettings, client redis.Cmdable) (StringCache, error) {
	switch cfg.Adapter {
	case "", "memory":
		return NewMemoryStringCache(cfg.MaxEntries)
	case "redis":
		return NewRedisStringCache(client, cfg.Prefix)
	case "memcache":
		return NewMemcacheStringCache(cfg.Host, cfg.Prefix)
	default:
		return nil, fmt.Errorf("unsupported cache adapter %q", cfg.Adapter)
	}
}
