package setting

// Database is the one database configuration used by Web and CLI commands.
var Database struct {
    DBSource string
    MigrationURL string
}

func LoadDBSetting() {
    loadDBSetting(CfgProvider)
}

func loadDBSetting(rootCfg ConfigProvider) {
    sec := rootCfg.Section("database")
    Database.DBSource = sec.Key("DB_SOURCE").String()
    Database.MigrationURL = sec.Key("MIGRATION_URL").String()
}
