package setting

import (
	"fmt"
	"strconv"
)

// DatabaseCommandSettings is the complete dependency configuration for
// PostgreSQL diagnostics. It does not depend on HTTP, Redis, OAuth or JWT.
type DatabaseCommandSettings struct {
	DBSource string
	Log LogGlobalConfig
}

func databaseCommandSettings(provider *ConfigProvider) (DatabaseCommandSettings, error) {
	settings := DatabaseCommandSettings{
		DBSource: provider.String("database", "DB_SOURCE"),
		Log: LoadLogSettings(provider),
	}
	if settings.DBSource == "" {
		return settings, fmt.Errorf("database.DB_SOURCE is required")
	}
	if err := settings.Log.Validate(); err != nil {
		return settings, err
	}
	return settings, nil
}

func LoadDatabaseCommandSettings(filename string) (DatabaseCommandSettings, error) {
	provider, err := InitCfgProvider(filename)
	if err != nil {
		return DatabaseCommandSettings{}, err
	}
	return databaseCommandSettings(provider)
}

// CleanupSettings adds only retention configuration to the PostgreSQL
// maintenance dependency set. Audit cleanup must remain possible when the
// public HTTP/OIDC configuration is broken or intentionally disabled.
type CleanupSettings struct {
	DatabaseCommandSettings
	AuditRetentionDays int
}

func LoadCleanupSettings(filename string) (CleanupSettings, error) {
	provider, err := InitCfgProvider(filename)
	if err != nil {
		return CleanupSettings{}, err
	}
	database, err := databaseCommandSettings(provider)
	if err != nil {
		return CleanupSettings{}, err
	}
	days, err := strconv.Atoi(provider.StringDefault("audit", "RETENTION_DAYS", "90"))
	if err != nil || days <= 0 || days > 3650 {
		return CleanupSettings{}, fmt.Errorf("audit RETENTION_DAYS must be between 1 and 3650")
	}
	return CleanupSettings{DatabaseCommandSettings: database, AuditRetentionDays: days}, nil
}
