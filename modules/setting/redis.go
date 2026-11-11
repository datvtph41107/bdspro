package setting

var Redis struct { Address string }

func loadRedisFrom(rootCfg ConfigProvider) {
    Redis.Address = rootCfg.Section("redis").Key("ADDR").String()
}

func LoadRedisSetting() {
    loadRedisFrom(CfgProvider)
}
