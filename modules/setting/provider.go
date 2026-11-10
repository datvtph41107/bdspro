package setting

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/ini.v1"
)

// ConfigProvider owns a read-only INI snapshot; env overrides are not
// persisted. Following Gitea, the parser stays inside modules/setting.
type ConfigProvider struct{ file *ini.File }

func InitCfgProvider(filename string) (*ConfigProvider, error) {
	f, err := ini.Load(filename)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", filename, err)
	}
	return &ConfigProvider{file: f}, nil
}

// Lookup distinguishes a missing option from an explicitly empty INI key or
// environment override. Use it for settings where an empty override is invalid.
func (p *ConfigProvider) Lookup(section, key string) (string, bool) {
	name := "BDS__" + strings.ToUpper(section) + "__" + strings.ToUpper(key)
	if section == "" {
		name = "BDS__" + strings.ToUpper(key)
	}
	if value, ok := os.LookupEnv(name); ok {
		return value, true
	}
	if p == nil || p.file == nil {
		return "", false
	}
	if section == "" {
		section = ini.DefaultSection
	}
	sec, err := p.file.GetSection(section)
	if err != nil {
		return "", false
	}
	k, err := sec.GetKey(key)
	if err != nil {
		return "", false
	}
	return k.String(), true
}

func (p *ConfigProvider) String(section, key string) string {
	value, _ := p.Lookup(section, key)
	return value
}
func (p *ConfigProvider) StringDefault(section, key, fallback string) string {
	if s := p.String(section, key); s != "" {
		return s
	}
	return fallback
}
