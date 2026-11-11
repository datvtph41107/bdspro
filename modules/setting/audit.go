package setting

import (
	"fmt"
	"strconv"
)

var Audit struct {
	HMACKey       string
	RetentionDays int
}

func loadAuditRetentionDays(rootCfg ConfigProvider) (int, error) {
	sec := rootCfg.Section("audit")
	days, err := strconv.Atoi(stringOrDefault(sec.Key("RETENTION_DAYS"), "90"))
	if err != nil || days < 1 || days > 3650 {
		return 0, fmt.Errorf("audit RETENTION_DAYS must be between 1 and 3650")
	}
	return days, nil
}

func LoadAuditRetentionDays() error {
	days, err := loadAuditRetentionDays(CfgProvider)
	if err != nil {
		return err
	}
	Audit.RetentionDays = days
	return nil
}

func loadAuditFrom(rootCfg ConfigProvider) error {
	days, err := loadAuditRetentionDays(rootCfg)
	if err != nil {
		return err
	}
	key := rootCfg.Section("audit").Key("HMAC_KEY").String()
	if key == "" && Common.RunMode == "development" {
		key = Security.TokenSymmetricKey
	}
	if len(key) < 32 {
		return fmt.Errorf("[audit] HMAC_KEY must contain at least 32 bytes")
	}
	if Common.RunMode != "development" && key == "abcdef0123456789abcdef0123456789" {
		return fmt.Errorf("production audit key must differ from development example")
	}
	Audit.RetentionDays, Audit.HMACKey = days, key
	return nil
}
