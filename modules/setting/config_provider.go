package setting

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/datvtph41107/bdspro/modules/util"
	"gopkg.in/ini.v1" // nolint:depguard // Chỉ adapter này làm việc trực tiếp với INI.
)

// ConfigKey cho đọc giá trị; IsSet phân biệt key thiếu với key rỗng.
type ConfigKey interface {
	Name() string
	String() string
	IsSet() bool
}

// ConfigSection chỉ cấp quyền đọc key thuộc đúng section.
type ConfigSection interface {
	Name() string
	Key(name string) ConfigKey
	Keys() []ConfigKey
}

// ConfigProvider là hợp đồng đọc của các loader, không lộ *ini.File.
type ConfigProvider interface {
	Section(name string) ConfigSection
}

type iniConfigProvider struct {
	file string
	ini  *ini.File
}

type iniConfigSection struct {
	sec *ini.Section
}

type iniConfigKey struct {
	name   string
	value  string
	exists bool
}

var (
	_ ConfigProvider = (*iniConfigProvider)(nil)
	_ ConfigSection  = (*iniConfigSection)(nil)
	_ ConfigKey      = (*iniConfigKey)(nil)
)

func (k *iniConfigKey) Name() string   { return k.name }
func (k *iniConfigKey) String() string { return k.value }
func (k *iniConfigKey) IsSet() bool    { return k.exists }

func (p *iniConfigProvider) Section(name string) ConfigSection {
	if name == "" {
		name = ini.DefaultSection
	}
	sec, err := p.ini.GetSection(name)
	if err != nil {
		return &iniConfigSection{}
	}
	return &iniConfigSection{sec: sec}
}

func (s *iniConfigSection) Name() string {
	if s.sec == nil {
		return ""
	}
	return s.sec.Name()
}

// Keys quét key trực tiếp để tránh kế thừa [parent] của gopkg.in/ini.v1.
func (s *iniConfigSection) Keys() (keys []ConfigKey) {
	if s.sec == nil {
		return nil
	}
	for _, k := range s.sec.Keys() {
		keys = append(keys, &iniConfigKey{name: k.Name(), value: k.String(), exists: true})
	}
	return keys
}

func (s *iniConfigSection) Key(name string) ConfigKey {
	if s.sec == nil {
		return &iniConfigKey{name: name}
	}
	// Gitea: duyệt Keys() trực tiếp, không lấy key từ section cha.
	for _, key := range s.sec.Keys() {
		if strings.EqualFold(key.Name(), name) {
			return &iniConfigKey{name: key.Name(), value: key.String(), exists: true}
		}
	}
	return &iniConfigKey{name: name}
}

// ConfigSectionKeyString tương ứng Gitea; không tạo key khi đọc giá trị mặc định.
func ConfigSectionKeyString(sec ConfigSection, key string, def ...string) string {
	if sec != nil {
		k := sec.Key(key)
		if k.IsSet() && k.String() != "" {
			return k.String()
		}
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

func configProviderLoadOptions() ini.LoadOptions {
	return ini.LoadOptions{
		KeyValueDelimiterOnWrite: " = ",
		IgnoreContinuation:       true,
	}
}

// NewConfigProviderFromData tạo provider trong RAM; dùng cùng loader options với file.
func NewConfigProviderFromData(configContent string) (ConfigProvider, error) {
	cfg, err := ini.LoadSources(configProviderLoadOptions(), strings.NewReader(configContent))
	if err != nil {
		return nil, fmt.Errorf("parse INI configuration: %w", err)
	}
	cfg.NameMapper = ini.SnackCase
	return &iniConfigProvider{ini: cfg}, nil
}

// NewConfigProviderFromFile yêu cầu app.ini có sẵn; không tự ghi file hay log.
func NewConfigProviderFromFile(file string) (ConfigProvider, error) {
	if file == "" {
		return nil, errors.New("configuration filename is empty")
	}

	cfg := ini.Empty(configProviderLoadOptions())
	isExist, err := util.IsExist(file)
	if err != nil {
		return nil, fmt.Errorf("unable to check if %q exists: %w", file, err)
	}
	if !isExist {
		return nil, fmt.Errorf("config file %q: %w", file, os.ErrNotExist)
	}
	if err = cfg.Append(file); err != nil {
		return nil, fmt.Errorf("failed to load config file %q: %w", file, err)
	}
	cfg.NameMapper = ini.SnackCase
	return &iniConfigProvider{file: file, ini: cfg}, nil
}

// set chỉ dùng khi bootstrap env; provider chưa được công bố.
func (p *iniConfigProvider) set(sectionName, keyName, value string) error {
	if sectionName == "" {
		sectionName = ini.DefaultSection
	}
	if keyName == "" {
		return errors.New("configuration key name is empty")
	}
	sec := p.ini.Section(sectionName)
	for _, key := range sec.Keys() {
		if strings.EqualFold(key.Name(), keyName) {
			key.SetValue(value)
			return nil
		}
	}
	if _, err := sec.NewKey(keyName, value); err != nil {
		return fmt.Errorf("set %s.%s: %w", sectionName, keyName, err)
	}
	return nil
}
