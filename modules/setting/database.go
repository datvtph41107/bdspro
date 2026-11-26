package setting

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

type DatabaseSettings struct {
	DBSource     string
	MigrationURL string
}

var Database DatabaseSettings

func loadDBSetting(cfg ConfigProvider, workPath string) (DatabaseSettings, error) {
	sec := cfg.Section("database")
	migrationURL, err := resolveMigrationURL(sec.Key("MIGRATION_URL").String(), workPath)
	if err != nil {
		return DatabaseSettings{}, err
	}
	return DatabaseSettings{
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
