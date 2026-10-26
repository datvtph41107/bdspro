package main

import (
    "github.com/datvtph41107/bdspro/bootstrap"
    "github.com/rs/zerolog/log"
)

func main() {
    if err := bootstrap.Run(); err != nil {
        log.Fatal().Err(err).Msg("BDSPro stopped")
    }
}

// Existing test entrypoint; migration implementation belongs to bootstrap.
func runDBMigration(migrationURL, dbSource string) error {
    return bootstrap.RunDBMigration(migrationURL, dbSource)
}
