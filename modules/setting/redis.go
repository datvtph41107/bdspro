package setting

type RedisSettings struct {
	Address string
}

var Redis RedisSettings

func loadRedisFrom(cfg ConfigProvider) RedisSettings {
	return RedisSettings{Address: cfg.Section("redis").Key("ADDR").String()}
}
