package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/urfave/cli/v3"
)

// doctor is deliberately read-only. Gitea has a richer services/doctor
// registry; a single BDSPro check does not justify copying that framework.
func newDoctorCommand() *cli.Command {
	return &cli.Command{
		Name:   "doctor",
		Usage:  "Detect organizations without any active owner (PostgreSQL only)",
		Before: PrepareConsoleLoggerLevel(log.INFO),
		Action: runDoctor,
	}
}

func runDoctor(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() != 0 {
		return fmt.Errorf("unexpected doctor argument %q", command.Args().First())
	}
	cfg, err := loadSettings(command)
	if err != nil {
		return err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	store, err := db.InitStore(checkCtx, cfg.DBSource)
	if err != nil {
		return fmt.Errorf("open doctor database: %w", err)
	}
	defer store.Close()

	report, err := organization_model.CheckOwnerCoverage(checkCtx, store)
	if err != nil {
		return fmt.Errorf("check organization owner coverage: %w", err)
	}
	if len(report.OrganizationIDs) == 0 {
		log.Info("Organization owner coverage: no ownerless organizations detected")
		return nil
	}
	if report.HasMore {
		return fmt.Errorf("organization owner coverage failed: at least %d ownerless organizations; first %d IDs: %v",
			len(report.OrganizationIDs)+1, len(report.OrganizationIDs), report.OrganizationIDs)
	}
	return fmt.Errorf("organization owner coverage failed: %d ownerless organizations: %v",
		len(report.OrganizationIDs), report.OrganizationIDs)
}
