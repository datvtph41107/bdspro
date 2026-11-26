package setting

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	AppWorkPath string
	CustomPath  string
	CustomConf  string
)

type PathOptions struct {
	WorkPath string
	Config   string
}

type Paths struct {
	WorkPath   string
	CustomPath string
	ConfigFile string
}

type stringWithDefault struct {
	Value string
	IsSet bool
}

func (s *stringWithDefault) Set(v string) {
	s.Value = v
	s.IsSet = true
}

func ResolvePaths(args PathOptions) (Paths, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Paths{}, fmt.Errorf("get current working directory: %w", err)
	}
	return resolvePaths(cwd, args)
}

func resolvePaths(cwd string, args PathOptions) (Paths, error) {
	if !filepath.IsAbs(cwd) {
		return Paths{}, errors.New("current working directory must be absolute")
	}

	tryAbsPath := func(paths ...string) string {
		s := paths[len(paths)-1]
		for i := len(paths) - 2; i >= 0; i-- {
			if filepath.IsAbs(s) {
				break
			}
			s = filepath.Join(paths[i], s)
		}
		return filepath.Clean(s)
	}

	tmpWorkPath := stringWithDefault{Value: cwd}
	tmpCustomConf := stringWithDefault{Value: "conf/app.ini"}

	readFromArgs := func() error {
		if args.WorkPath != "" {
			tmpWorkPath.Set(args.WorkPath)
			if !filepath.IsAbs(tmpWorkPath.Value) {
				return fmt.Errorf("--work-path must be absolute path: %q", tmpWorkPath.Value)
			}
		}
		if args.Config != "" {
			tmpCustomConf.Set(args.Config)
			if !filepath.IsAbs(tmpCustomConf.Value) {

				tmpCustomConf.Set(tryAbsPath(cwd, tmpCustomConf.Value))
			}
		}
		return nil
	}

	if err := readFromArgs(); err != nil {
		return Paths{}, err
	}

	customPath := tryAbsPath(tmpWorkPath.Value, "custom")
	if !tmpCustomConf.IsSet {
		tmpCustomConf.Set(tryAbsPath(customPath, tmpCustomConf.Value))
	}
	return Paths{
		WorkPath:   filepath.Clean(tmpWorkPath.Value),
		CustomPath: customPath,
		ConfigFile: filepath.Clean(tmpCustomConf.Value),
	}, nil
}

func samePath(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	fiA, errA := os.Stat(a)
	fiB, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(fiA, fiB)
}
