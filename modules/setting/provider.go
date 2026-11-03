package setting

import (
    "fmt"
    "os"
    "strings"

    "gopkg.in/ini.v1"
)

// ConfigProvider owns a read-only INI snapshot; env overrides are not
// persisted. Following Gitea, the parser stays inside modules/setting.
type ConfigProvider struct { file *ini.File }

func InitCfgProvider(filename string) (*ConfigProvider, error) {
    f, err := ini.Load(filename)
    if err != nil { return nil, fmt.Errorf("read config %q: %w",filename,err) }
    return &ConfigProvider{file:f}, nil
}
func (p *ConfigProvider) String(section, key string) string {
    name := "BDS__" + strings.ToUpper(section) + "__" + strings.ToUpper(key)
    if value, ok := os.LookupEnv(name); ok { return value }
    if section == "" { section = ini.DefaultSection }
    sec, err := p.file.GetSection(section)
    if err != nil { return "" }
    k, err := sec.GetKey(key)
    if err != nil { return "" }
    return k.String()
}
func (p *ConfigProvider) StringDefault(section,key,fallback string) string {
    if s := p.String(section,key); s != "" { return s }
    return fallback
}
