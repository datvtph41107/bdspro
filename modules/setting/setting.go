package setting

import "time"

// Process metadata is set from the root main.go before CLI execution.
var (
	AppVer       string
	AppBuiltWith string
	AppStartTime time.Time
	CustomConf   string
)
