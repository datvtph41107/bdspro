package setting

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

// DatabaseSettings giữ nguồn DB và migrations; không mở connection.
type DatabaseSettings struct {
	DBSource     string `ini:"DB_SOURCE"`
	MigrationURL string `ini:"MIGRATION_URL"`
}

var Database DatabaseSettings

// loadDBSetting chuẩn hóa migration path theo AppWorkPath, không theo shell CWD.
// Migrate dùng cùng instance root như logger khi chạy binary ngoài thư mục dự án.
func loadDBSetting(cfg ConfigProvider, workPath string) (DatabaseSettings, error) {
	var result DatabaseSettings
	if err := mapSettings(cfg, "database", &result); err != nil {
		return DatabaseSettings{}, err
	}
	if strings.TrimSpace(result.DBSource) == "" {
		return DatabaseSettings{}, errors.New("database.DB_SOURCE is required")
	}
	migrationURL, err := resolveMigrationURL(result.MigrationURL, workPath)
	if err != nil {
		return DatabaseSettings{}, err
	}
	result.MigrationURL = migrationURL
	return result, nil
}

// resolveMigrationURL chỉ xử lý file://; các scheme khác được giữ nguyên.
func resolveMigrationURL(raw, workPath string) (string, error) {
	if raw == "" || !strings.HasPrefix(raw, "file://") {
		return raw, nil
	}
	if workPath == "" || !filepath.IsAbs(workPath) {
		return "", errors.New("database.MIGRATION_URL requires an absolute work path")
	}

	// Gitea: đường dẫn resource tương đối cần được gắn với AppWorkPath.
	// file://db/migration có đường dẫn tương đối "db/migration".
	path := strings.TrimPrefix(raw, "file://")
	if path == "" {
		return "", errors.New("database.MIGRATION_URL file path is empty")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(workPath, path)
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Clean(path))}).String(), nil
}

// MigrateConfig contains only the settings needed by the migrate command.
type MigrateConfig struct {
	Paths    Paths
	Common   CommonSettings
	Database DatabaseSettings
}

// LoadMigrateConfig validates common and database settings without publishing
// globals or constructing HTTP/Redis/Auth resources.
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

// LoadMigrateSettings publishes only the common and DB settings needed
// by a migration command; Web/Redis/Auth are not initialized.
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
