package setting

// RedisSettings chỉ chứa địa chỉ; kết nối do modules/conn sở hữu.
type RedisSettings struct {
	Address string
}

var Redis RedisSettings

func loadRedisFrom(cfg ConfigProvider) RedisSettings {
	return RedisSettings{Address: cfg.Section("redis").Key("ADDR").String()}
}
