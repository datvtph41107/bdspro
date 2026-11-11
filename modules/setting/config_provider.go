package setting

import (
	"fmt"
	"strings"

	"gopkg.in/ini.v1"
)

// ConfigKey represents a single configuration value.
// It distinguishes a missing key from an explicitly
// configured empty value.
type ConfigKey interface {
	String() string
	IsSet() bool
}

// ConfigSection provides access to keys belonging
// to a configuration section.
type ConfigSection interface {
	Key(name string) ConfigKey
}

// ConfigProvider is the read-only configuration
// contract consumed by domain setting loaders.
type ConfigProvider interface {
	Section(name string) ConfigSection
}

// iniConfigProvider owns the underlying INI file.
// Domain loaders never access ini.File directly.
type iniConfigProvider struct {
	file *ini.File
}

// iniConfigSection wraps one INI section.
// A nil section represents a missing section.
type iniConfigSection struct {
	section *ini.Section
}

// iniConfigKey contains a snapshot of a key.
// Domain loaders cannot mutate the INI parser.
type iniConfigKey struct {
	value  string
	exists bool
}

var (
	_ ConfigProvider = (*iniConfigProvider)(nil)
	_ ConfigSection  = (*iniConfigSection)(nil)
	_ ConfigKey      = (*iniConfigKey)(nil)
)

// String returns the key's value.
// A missing key returns an empty string.
func (k *iniConfigKey) String() string {
	return k.value
}

// IsSet reports whether the key was explicitly
// present, including when its value was empty.
func (k *iniConfigKey) IsSet() bool {
	return k.exists
}

// Section returns a section accessor.
// A missing section returns an empty accessor,
// so callers can safely read optional settings.
func (p *iniConfigProvider) Section(
	name string,
) ConfigSection {
	if name == "" {
		name = ini.DefaultSection
	}

	section, err := p.file.GetSection(name)
	if err != nil {
		return &iniConfigSection{}
	}

	return &iniConfigSection{
		section: section,
	}
}

// Key searches only keys belonging to this section.
// It deliberately avoids inherited parent-section
// values returned by some ini.Section.Key calls.
func (s *iniConfigSection) Key(
	name string,
) ConfigKey {
	if s.section == nil {
		return &iniConfigKey{}
	}

	for _, key := range s.section.Keys() {
		if strings.EqualFold(key.Name(), name) {
			return &iniConfigKey{
				value:  key.String(),
				exists: true,
			}
		}
	}

	return &iniConfigKey{}
}

// configProviderLoadOptions keeps INI parsing
// behavior centralized.
func configProviderLoadOptions() ini.LoadOptions {
	return ini.LoadOptions{
		IgnoreContinuation: true,
	}
}

// NewConfigProviderFromFile loads an installed
// application's configuration.
//
// BDSPro currently requires an existing app.ini.
// Unlike Gitea's installer flow, a missing file
// is an error.
func NewConfigProviderFromFile(
	filename string,
) (ConfigProvider, error) {
	if filename == "" {
		return nil, fmt.Errorf(
			"configuration filename is empty",
		)
	}

	file, err := ini.LoadSources(
		configProviderLoadOptions(),
		filename,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"load config %q: %w",
			filename, err,
		)
	}

	return &iniConfigProvider{
		file: file,
	}, nil
}

// NewConfigProviderFromData creates a provider
// from INI content, primarily for unit tests.
func NewConfigProviderFromData(
	content string,
) (ConfigProvider, error) {
	file, err := ini.LoadSources(
		configProviderLoadOptions(),
		strings.NewReader(content),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"parse INI configuration: %w", err,
		)
	}

	return &iniConfigProvider{
		file: file,
	}, nil
}

// set updates the effective configuration
// during initialization only.
//
// It is intentionally NOT part of ConfigProvider.
// Domain loaders receive a read-only interface.
func (p *iniConfigProvider) set(
	sectionName, keyName, value string,
) error {
	if sectionName == "" {
		sectionName = ini.DefaultSection
	}

	if keyName == "" {
		return fmt.Errorf(
			"configuration key name is empty",
		)
	}

	section := p.file.Section(sectionName)

	// Avoid inherited-key updates.
	for _, key := range section.Keys() {
		if strings.EqualFold(key.Name(), keyName) {
			key.SetValue(value)
			return nil
		}
	}

	if _, err := section.NewKey(
		keyName, value,
	); err != nil {
		return fmt.Errorf(
			"set %s.%s: %w",
			sectionName, keyName, err,
		)
	}

	return nil
}
