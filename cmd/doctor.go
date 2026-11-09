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
		Flags: []cli.Flag{
			&cli.Int64Flag{Name: "after-id", Usage: "Inspect only organizations after this party ID (default: 0)"},
		},
		Action: runDoctor,
	}
}

func runDoctor(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() != 0 {
		return fmt.Errorf("unexpected doctor argument %q", command.Args().First())
	}
	afterID := command.Int64("after-id")
	if afterID < 0 {
		return fmt.Errorf("--after-id cannot be negative")
	}
	cfg, err := loadDatabaseCommandSettings(command)
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

	report, err := organization_model.CheckOwnerCoverage(checkCtx, store, afterID)
	if err != nil {
		return fmt.Errorf("check organization owner coverage: %w", err)
	}
	if len(report.OrganizationIDs) == 0 {
		log.Info("Organization owner coverage: none detected after party ID %d", afterID)
		return nil
	}
	if report.HasMore {
		return fmt.Errorf("organization owner coverage failed: at least %d ownerless organizations after party ID %d; first %d IDs: %v; continue with doctor --after-id %d",
			len(report.OrganizationIDs)+1, afterID, len(report.OrganizationIDs), report.OrganizationIDs,
			report.OrganizationIDs[len(report.OrganizationIDs)-1])
	}
	return fmt.Errorf("organization owner coverage failed: %d ownerless organizations after party ID %d: %v",
		len(report.OrganizationIDs), afterID, report.OrganizationIDs)
}
