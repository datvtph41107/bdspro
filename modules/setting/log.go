package setting

import "github.com/datvtph41107/bdspro/modules/log"

type LogGlobalConfig struct {
	Mode, Format, Level, RootPath, FileName string
}

func InitLoggers(c LogGlobalConfig) error {
	return log.GetManager().ConfigureOutputs(c.Mode, c.Format, c.Level, c.RootPath, c.FileName)
}
