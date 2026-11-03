package main

import (
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/cmd"
	"github.com/datvtph41107/bdspro/modules/log"
	"github.com/datvtph41107/bdspro/modules/setting"
	"github.com/urfave/cli/v3"
)

var (
	Version = "development"
	Tags    = ""
)

func init() {
	setting.AppVer = Version
	setting.AppBuiltWith = formatBuiltWith()
	setting.AppStartTime = time.Now().UTC()
}

// @title BDSPro API
// @version 1.0
// @description BDSPro REST API.
// @BasePath /
func main() {
	cli.OsExiter = func(code int) {
		log.GetManager().Close()
		os.Exit(code)
	}
	app := cmd.NewMainApp(cmd.AppVersion{Version: Version, Extra: formatBuiltWith()})
	_ = cmd.RunMainApp(app, os.Args...)
	log.GetManager().Close()
}
func formatBuiltWith() string {
	version := runtime.Version()
	if len(Tags) == 0 {
		return " built with " + version
	}
	return " built with " + version + " : " + strings.ReplaceAll(Tags, " ", ", ")
}
