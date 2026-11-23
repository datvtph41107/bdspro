package setting

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

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
	MapTo(dst any) error
}

// ConfigProvider là hợp đồng đọc của các loader, không lộ *ini.File.
type ConfigProvider interface {
	Section(name string) ConfigSection
	Sections() []ConfigSection
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

// Sections exposes all INI sections so validation can detect unknown keys.
func (p *iniConfigProvider) Sections() []ConfigSection {
	sections := p.ini.Sections()
	result := make([]ConfigSection, 0, len(sections))
	for _, section := range sections {
		result = append(result, &iniConfigSection{sec: section})
	}
	return result
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
	// This adapter owns opening the config file; a one-use modules/util
	// wrapper around os.Stat only hid the actual filesystem dependency.
	if _, err := os.Stat(file); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file %q: %w", file, os.ErrNotExist)
		}
		return nil, fmt.Errorf("unable to check config file %q: %w", file, err)
	}
	if err := cfg.Append(file); err != nil {
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

// MapTo only reads keys physically present in this section. The upstream INI
// mapper can inherit parent values and silently skip invalid conversions,
// so settings use this strict section-local scalar mapper instead.
func (s *iniConfigSection) MapTo(dst any) error {
	v := reflect.ValueOf(dst)
	if !v.IsValid() || v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return errors.New("config mapping requires a non-nil pointer to a struct")
	}
	value := v.Elem()
	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		fieldType := typ.Field(i)
		field := value.Field(i)
		if !field.CanSet() {
			continue
		}
		name := fieldType.Tag.Get("ini")
		if name == "-" {
			continue
		}
		if name == "" {
			name = ini.SnackCase(fieldType.Name)
		}
		key := s.Key(name)
		if !key.IsSet() {
			continue
		}
		if err := setConfigScalar(field, key.String()); err != nil {
			return fmt.Errorf("invalid configuration [%s].%s: %w", s.Name(), name, err)
		}
	}
	return nil
}

func setConfigScalar(field reflect.Value, raw string) error {
	if field.Type() == reflect.TypeOf(time.Duration(0)) {
		value, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		field.SetInt(int64(value))
		return nil
	}
	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Bool:
		value, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return fmt.Errorf("invalid boolean: %w", err)
		}
		field.SetBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid integer: %w", err)
		}
		field.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid unsigned integer: %w", err)
		}
		field.SetUint(value)
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(strings.TrimSpace(raw), field.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid floating point: %w", err)
		}
		field.SetFloat(value)
	default:
		return fmt.Errorf("unsupported config field type %s", field.Type())
	}
	return nil
}

// mapSettings is the shared mapping primitive for subsystem loaders.
func mapSettings(cfg ConfigProvider, section string, dst any) error {
	if err := cfg.Section(section).MapTo(dst); err != nil {
		return fmt.Errorf("map section [%s]: %w", section, err)
	}
	return nil
}
