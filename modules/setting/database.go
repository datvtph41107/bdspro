package setting

import "fmt"

// DatabaseSettings limits migration initialization to its actual dependencies.
type DatabaseSettings struct {
	Environment  string
	DBSource     string
	MigrationURL string
}

func LoadDatabaseSettings(filename string) (DatabaseSettings, error) {
	p, err := InitCfgProvider(filename)
	if err != nil {
		return DatabaseSettings{}, err
	}
	c := DatabaseSettings{
		Environment:  p.StringDefault("", "RUN_MODE", "development"),
		DBSource:     p.String("database", "DB_SOURCE"),
		MigrationURL: p.String("database", "MIGRATION_URL"),
	}
	if c.DBSource == "" || c.MigrationURL == "" {
		return c, fmt.Errorf("database.DB_SOURCE and database.MIGRATION_URL are required")
	}
	return c, nil
}
