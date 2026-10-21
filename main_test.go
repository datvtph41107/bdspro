package main

import (
	"os"
	"testing"

	"github.com/datvtph41107/bdspro/util"
	"github.com/stretchr/testify/require"
)

// The Actions runner owns a disposable database, unlike the developer's local bdspro.
func TestRunDBMigrationNoChange(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("integration test requires the disposable CI database")
	}
	config, err := util.LoadConfig(".")
	require.NoError(t, err)
	require.NoError(t, runDBMigration(config.MigrationURL, config.DBSource))
}
