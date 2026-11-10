package setting

import "fmt"

// DatabaseSettings limits migration initialization to its actual dependencies.
type DatabaseSettings struct {
	DatabaseCommandSettings
	Environment  string
	MigrationURL string
}

func LoadDatabaseSettings(filename string) (DatabaseSettings, error) {
	p, err := InitCfgProvider(filename)
	if err != nil {
		return DatabaseSettings{}, err
	}
	base, err := databaseCommandSettings(p)
	if err != nil {
		return DatabaseSettings{}, err
	}
	c := DatabaseSettings{
		DatabaseCommandSettings: base,
		Environment:             p.String("", "RUN_MODE"),
		MigrationURL:            p.String("database", "MIGRATION_URL"),
	}
	if c.MigrationURL == "" {
		return c, fmt.Errorf("database.MIGRATION_URL is required")
	}
	return c, nil
}
