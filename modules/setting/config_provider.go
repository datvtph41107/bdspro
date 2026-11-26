package setting

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/ini.v1"
)

type ConfigKey interface {
	Name() string
	String() string
	IsSet() bool
}

type ConfigSection interface {
	Name() string
	Key(name string) ConfigKey
	Keys() []ConfigKey
}

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

	for _, key := range s.sec.Keys() {
		if strings.EqualFold(key.Name(), name) {
			return &iniConfigKey{name: key.Name(), value: key.String(), exists: true}
		}
	}
	return &iniConfigKey{name: name}
}

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

func NewConfigProviderFromData(configContent string) (ConfigProvider, error) {
	cfg, err := ini.LoadSources(configProviderLoadOptions(), strings.NewReader(configContent))
	if err != nil {
		return nil, fmt.Errorf("parse INI configuration: %w", err)
	}
	cfg.NameMapper = ini.SnackCase
	return &iniConfigProvider{ini: cfg}, nil
}

func NewConfigProviderFromFile(file string) (ConfigProvider, error) {
	if file == "" {
		return nil, errors.New("configuration filename is empty")
	}

	cfg := ini.Empty(configProviderLoadOptions())

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
