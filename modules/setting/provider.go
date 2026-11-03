package setting

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// ConfigProvider is a read-only view of one INI file plus explicit env overrides.
type ConfigProvider struct{ file *viper.Viper }

func InitCfgProvider(filename string) (*ConfigProvider, error) {
	v := viper.New()
	v.SetConfigFile(filename)
	v.SetConfigType("ini")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %q: %w", filename, err)
	}
	return &ConfigProvider{file: v}, nil
}
func (p *ConfigProvider) String(section, key string) string {
	name := "BDS__" + strings.ToUpper(section) + "__" + strings.ToUpper(key)
	if s, ok := os.LookupEnv(name); ok {
		return s
	}
	if section == "" {
		return p.file.GetString(key)
	}
	return p.file.GetString(section + "." + key)
}
func (p *ConfigProvider) StringDefault(section, key, fallback string) string {
	if s := p.String(section, key); s != "" {
		return s
	}
	return fallback
}
