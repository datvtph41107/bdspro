package cmd

import (
	"fmt"
	"strings"

	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
)

// configureCommandLogger is a process bootstrap action, not a setting loader.
// Modules/setting validates options; modules/log owns the actual writer.
func configureCommandLogger(cfg setting.LogGlobalConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := log.GetManager().ConfigureOutputs(
		cfg.Mode, cfg.Format, strings.ToLower(cfg.Level), cfg.RootPath, cfg.FileName,
	); err != nil {
		return fmt.Errorf("configure logging outputs: %w", err)
	}
	return nil
}
