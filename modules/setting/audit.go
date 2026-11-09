package setting

import (
	"fmt"
	"strconv"
)

// LoadAuditRetentionDays is the single validation rule used by web, worker
// and one-shot cleanup. The retention policy is a database lifecycle rule,
// not an HTTP or queue-specific setting.
func LoadAuditRetentionDays(provider *ConfigProvider) (int, error) {
	days, err := strconv.Atoi(provider.StringDefault("audit", "RETENTION_DAYS", "90"))
	if err != nil || days <= 0 || days > 3650 {
		return 0, fmt.Errorf("audit RETENTION_DAYS must be between 1 and 3650")
	}
	return days, nil
}
