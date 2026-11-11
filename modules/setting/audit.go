package setting

import (
    "fmt"
    "strconv"
)

var Audit struct {
    HMACKey string
    RetentionDays int
}

func LoadAuditRetentionDays() error {
    sec := CfgProvider.Section("audit")
    days, err := strconv.Atoi(stringOrDefault(sec.Key("RETENTION_DAYS"), "90"))
    if err != nil || days < 1 || days > 3650 {
        return fmt.Errorf("audit RETENTION_DAYS must be between 1 and 3650")
    }
    Audit.RetentionDays = days
    return nil
}

func loadAuditFrom(rootCfg ConfigProvider) error {
    if err := LoadAuditRetentionDays(); err != nil { return err }
    key := rootCfg.Section("audit").Key("HMAC_KEY").String()
    if key == "" && Common.RunMode == "development" {
        key = Security.TokenSymmetricKey
    }
    if len(key) < 32 { return fmt.Errorf("[audit] HMAC_KEY must contain at least 32 bytes") }
    if Common.RunMode != "development" && key == "abcdef0123456789abcdef0123456789" {
        return fmt.Errorf("production audit key must differ from development example")
    }
    Audit.HMACKey = key
    return nil
}
