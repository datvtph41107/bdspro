package setting

import (
	"errors"
	"strings"
)

// RedisSettings chỉ chứa địa chỉ; kết nối do modules/conn sở hữu.
type RedisSettings struct {
	Address string `ini:"ADDR"`
}

var Redis RedisSettings

func loadRedisFrom(cfg ConfigProvider) (RedisSettings, error) {
	var result RedisSettings
	if err := mapSettings(cfg, "redis", &result); err != nil {
		return RedisSettings{}, err
	}
	if strings.TrimSpace(result.Address) == "" {
		return RedisSettings{}, errors.New("redis.ADDR is required")
	}
	return result, nil
}
