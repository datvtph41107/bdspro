package setting

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CacheSettings configures an optional cache, independently of Redis security
// state and of the Asynq queue. Cache is never initialized on API startup
// until a caller explicitly needs it.
type CacheSettings struct {
	Adapter    string
	Host       string
	Prefix     string
	MaxEntries int
	DefaultTTL time.Duration
}

func LoadCacheSettings(provider *ConfigProvider) (CacheSettings, error) {
	if provider == nil {
		return CacheSettings{}, fmt.Errorf("cache configuration provider is nil")
	}
	cfg := CacheSettings{
		Adapter:    strings.ToLower(provider.StringDefault("cache", "ADAPTER", "memory")),
		Host:       provider.String("cache", "HOST"),
		Prefix:     provider.StringDefault("cache", "PREFIX", "cache:"),
		MaxEntries: 10000,
		DefaultTTL: 5 * time.Minute,
	}
	switch cfg.Adapter {
	case "memory", "redis", "memcache":
	default:
		return CacheSettings{}, fmt.Errorf("unsupported cache adapter %q", cfg.Adapter)
	}
	if raw := provider.String("cache", "MAX_ENTRIES"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return CacheSettings{}, fmt.Errorf("cache MAX_ENTRIES must be positive")
		}
		cfg.MaxEntries = n
	}
	if raw := provider.String("cache", "ITEM_TTL"); raw != "" {
		ttl, err := time.ParseDuration(raw)
		if err != nil || ttl <= 0 {
			return CacheSettings{}, fmt.Errorf("cache ITEM_TTL must be a positive duration: %q", raw)
		}
		cfg.DefaultTTL = ttl
	}
	if cfg.Adapter == "memcache" && cfg.Host == "" {
		return CacheSettings{}, fmt.Errorf("Memcached HOST is required")
	}
	if cfg.Prefix == "" {
		return CacheSettings{}, fmt.Errorf("cache PREFIX is required")
	}
	return cfg, nil
}
