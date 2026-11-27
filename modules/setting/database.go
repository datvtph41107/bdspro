package setting

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

type DatabaseSettings struct {
	DBSource           string
	MigrationURL       string
	LogSQL             string
	SlowQueryThreshold time.Duration
}

var Database DatabaseSettings

func loadDBSetting(cfg ConfigProvider, workPath string) (DatabaseSettings, error) {
	sec := cfg.Section("database")
	migrationURL, err := resolveMigrationURL(sec.Key("MIGRATION_URL").String(), workPath)
	if err != nil {
		return DatabaseSettings{}, err
	}
	mode := stringOrDefault(sec.Key("LOG_SQL"), "slow")
	if mode != "off" && mode != "slow" && mode != "all" {
		return DatabaseSettings{}, errors.New("database.LOG_SQL must be off, slow or all")
	}
	threshold, err := time.ParseDuration(stringOrDefault(sec.Key("SLOW_QUERY_THRESHOLD"), "200ms"))
	if err != nil || threshold <= 0 {
		return DatabaseSettings{}, fmt.Errorf("database.SLOW_QUERY_THRESHOLD must be a positive duration")
	}
	return DatabaseSettings{
		LogSQL: mode, SlowQueryThreshold: threshold,
		DBSource:     sec.Key("DB_SOURCE").String(),
		MigrationURL: migrationURL,
	}, nil
}

func resolveMigrationURL(raw, workPath string) (string, error) {
	if raw == "" || !strings.HasPrefix(raw, "file://") {
		return raw, nil
	}
	if workPath == "" || !filepath.IsAbs(workPath) {
		return "", errors.New("database.MIGRATION_URL requires an absolute work path")
	}

	path := strings.TrimPrefix(raw, "file://")
	if path == "" {
		return "", errors.New("database.MIGRATION_URL file path is empty")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(workPath, path)
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Clean(path))}).String(), nil
}

type MigrateConfig struct {
	Paths    Paths
	Common   CommonSettings
	Database DatabaseSettings
}

func LoadMigrateConfig(args PathOptions, envs []string) (MigrateConfig, error) {
	prepared, err := prepareCommonConfig(args, envs)
	if err != nil {
		return MigrateConfig{}, err
	}
	database, err := loadDBSetting(prepared.provider, prepared.paths.WorkPath)
	if err != nil {
		return MigrateConfig{}, err
	}
	return MigrateConfig{
		Paths: prepared.paths, Common: prepared.common, Database: database,
	}, nil
}

func LoadMigrateSettings(args PathOptions, envs []string) error {
	cfg, err := LoadMigrateConfig(args, envs)
	if err != nil {
		return err
	}
	return publishSettings(func() {
		publishCommonSettings(cfg.Paths, cfg.Common)
		Database = cfg.Database
	})
}
