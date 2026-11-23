package setting

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Path globals được publish sau validation; đặt tại path.go như Gitea.
var (
	AppWorkPath string
	CustomPath  string
	CustomConf  string
)

// PathOptions chứa override CLI; chuỗi rỗng nghĩa là không override.
type PathOptions struct {
	WorkPath string
	Config   string
}

// Paths là kết quả đã chuẩn hóa; chưa công bố ra globals.
type Paths struct {
	WorkPath   string
	CustomPath string
	ConfigFile string
}

// stringWithDefault giữ trạng thái có override, cùng pattern với Gitea.
type stringWithDefault struct {
	Value string
	IsSet bool
}

func (s *stringWithDefault) Set(v string) {
	s.Value = v
	s.IsSet = true
}

// ResolvePaths chỉ lấy CWD; thuật toán nằm ở resolvePaths để dễ kiểm thử.
func ResolvePaths(args PathOptions) (Paths, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Paths{}, fmt.Errorf("get current working directory: %w", err)
	}
	return resolvePaths(cwd, args)
}

// resolvePaths làm như Gitea: default -> overrides -> resolve.
// BDSPro giữ CWD làm default thay vì thư mục binary (hợp đồng deploy).
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
				// Explicit --config tương đối theo CWD, không theo WorkPath.
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

// samePath kiểm tra đồng nhất thực thể, kể cả symlink nếu cả hai tồn tại.
func samePath(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	fiA, errA := os.Stat(a)
	fiB, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(fiA, fiB)
}

// resolveConfiguredWorkPath finishes path resolution after config/ENV loading.
// The config file location was resolved first, so WORK_PATH never silently
// changes which file is being read.
func resolveConfiguredWorkPath(paths Paths, args PathOptions, cfg ConfigProvider) (Paths, error) {
	configured := ConfigSectionKeyString(cfg.Section(""), "WORK_PATH")
	if configured == "" {
		return paths, nil
	}
	if !filepath.IsAbs(configured) {
		return Paths{}, fmt.Errorf("WORK_PATH in %q must be absolute path: %q", paths.ConfigFile, configured)
	}
	configured = filepath.Clean(configured)
	if args.WorkPath != "" && !samePath(paths.WorkPath, configured) {
		return Paths{}, fmt.Errorf("WORK_PATH %q conflicts with --work-path %q", configured, paths.WorkPath)
	}
	paths.WorkPath = configured
	paths.CustomPath = filepath.Join(configured, "custom")
	return paths, nil
}
