package setting

import "fmt"

// LoadWorkerSettings loads only the settings consumed by the worker.
// The CLI must prepare the canonical ConfigProvider and Common first.
func LoadWorkerSettings() error {
	if CfgProvider == nil {
		return fmt.Errorf("configuration provider is not initialized")
	}
	loadDBSetting(CfgProvider)
	loadRedisFrom(CfgProvider)
	return loadQueueFrom(CfgProvider)
}
