package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/datvtph41107/bdspro/models/db"
	organization_model "github.com/datvtph41107/bdspro/models/organization"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/urfave/cli/v3"
)

func newDoctorCommand() *cli.Command {
	return &cli.Command{
		Name: "doctor", Usage: "Check organizations without an active owner",
		Flags: []cli.Flag{
			&cli.Int64Flag{Name: "after-id", Usage: "Inspect organizations after this party ID"},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			if next, err := rejectArguments("doctor")(ctx, c); err != nil {
				return next, err
			}
			if c.Int64("after-id") < 0 {
				return ctx, fmt.Errorf("--after-id cannot be negative")
			}
			return ctx, nil
		},
		Action: runDoctor,
	}
}

func runDoctor(ctx context.Context, command *cli.Command) error {
	cfg, err := setting.LoadDoctorSettings()
	if err != nil {
		return fmt.Errorf("load doctor settings: %w", err)
	}
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	store, err := db.InitStore(checkCtx, cfg.DBSource)
	if err != nil {
		return fmt.Errorf("open doctor database: %w", err)
	}
	defer store.Close()

	afterID := command.Int64("after-id")
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
