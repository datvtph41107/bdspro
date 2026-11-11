package setting

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	EnvConfigKeyPrefix     = "BDS__"
	EnvConfigKeySuffixFile = "__FILE"
)

// Gitea-style encoding for characters that
// cannot appear directly in environment names.
//
// Example: log_0X2E_file -> log.file
var envEscapedBytes = regexp.MustCompile(
	`_0[xX](?:[0-9a-fA-F]{2})+_`,
)

// configOverride is an effective value prepared
// before the configuration provider is changed.
type configOverride struct {
	section string
	key     string
	value   string
}

// decodeEnvPart decodes escaped UTF-8 bytes
// in an environment section or key name.
func decodeEnvPart(encoded string) (string, error) {
	var result strings.Builder
	start := 0

	for _, match := range envEscapedBytes.FindAllStringIndex(
		encoded, -1,
	) {
		result.WriteString(encoded[start:match[0]])

		escape := encoded[match[0]:match[1]]
		data, err := hex.DecodeString(
			escape[3 : len(escape)-1],
		)
		if err != nil {
			return "", fmt.Errorf(
				"invalid environment escape: %w",
				err,
			)
		}

		result.Write(data)
		start = match[1]
	}

	result.WriteString(encoded[start:])

	if !utf8.ValidString(result.String()) {
		return "", fmt.Errorf(
			"environment key contains invalid UTF-8",
		)
	}

	return result.String(), nil
}

// decodeEnvironmentKey accepts:
//
// BDS__SECTION__KEY
// BDS__SECTION__KEY__FILE
// BDS____KEY
//
// The last form addresses the default INI section.
func decodeEnvironmentKey(
	name string,
) (
	section string,
	key string,
	fromFile bool,
	ok bool,
) {
	if !strings.HasPrefix(
		name, EnvConfigKeyPrefix,
	) {
		return "", "", false, false
	}

	encoded := strings.TrimPrefix(
		name, EnvConfigKeyPrefix,
	)

	if strings.HasSuffix(
		encoded, EnvConfigKeySuffixFile,
	) {
		fromFile = true
		encoded = strings.TrimSuffix(
			encoded, EnvConfigKeySuffixFile,
		)
	}

	rawSection, rawKey, found := strings.Cut(
		encoded, "__",
	)
	if !found || rawKey == "" ||
		strings.Contains(rawKey, "__") {
		return "", "", fromFile, false
	}

	section, err := decodeEnvPart(rawSection)
	if err != nil {
		return "", "", fromFile, false
	}

	key, err = decodeEnvPart(rawKey)
	if err != nil || key == "" {
		return "", "", fromFile, false
	}

	return strings.ToLower(section),
		key, fromFile, true
}

// readEnvironmentFile reads a secret value.
// Only one trailing line ending is removed,
// matching Gitea's __FILE behavior.
func readEnvironmentFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf(
			"secret file path is empty",
		)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf(
			"read secret file: %w", err,
		)
	}

	switch {
	case bytes.HasSuffix(
		content, []byte("\r\n"),
	):
		content = content[:len(content)-2]

	case bytes.HasSuffix(
		content, []byte("\n"),
	):
		content = content[:len(content)-1]
	}

	if !utf8.Valid(content) {
		return "", fmt.Errorf(
			"secret file is not valid UTF-8",
		)
	}

	return string(content), nil
}

// collectEnvironmentOverrides validates and
// prepares every override before mutation.
func collectEnvironmentOverrides(
	envs []string,
) ([]configOverride, error) {
	overrides := make([]configOverride, 0)
	seen := make(map[string]struct{})

	for _, entry := range envs {
		name, value, found := strings.Cut(
			entry, "=",
		)
		if !found || !strings.HasPrefix(
			name, EnvConfigKeyPrefix,
		) {
			continue
		}

		section, key, fromFile, valid :=
			decodeEnvironmentKey(name)

		if !valid {
			return nil, fmt.Errorf(
				"invalid config environment name %q",
				name,
			)
		}

		identity := strings.ToLower(section) +
			"\x00" + strings.ToUpper(key)

		if _, exists := seen[identity]; exists {
			return nil, fmt.Errorf(
				"duplicate configuration override %q",
				name,
			)
		}

		seen[identity] = struct{}{}

		if fromFile {
			secret, err := readEnvironmentFile(
				value,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"environment %s: %w",
					name, err,
				)
			}
			value = secret
		}

		overrides = append(
			overrides,
			configOverride{
				section: section,
				key:     key,
				value:   value,
			},
		)
	}

	return overrides, nil
}

// applyEnvironmentOverrides adds process
// overrides to the effective INI provider.
//
// This function must run before CfgProvider
// is published to application consumers.
func applyEnvironmentOverrides(
	provider ConfigProvider,
	envs []string,
) error {
	writable, ok := provider.(*iniConfigProvider)
	if !ok || writable == nil {
		return fmt.Errorf(
			"configuration provider does not support overrides",
		)
	}

	overrides, err := collectEnvironmentOverrides(envs)
	if err != nil {
		return err
	}

	for _, override := range overrides {
		if err := writable.set(
			override.section,
			override.key,
			override.value,
		); err != nil {
			return fmt.Errorf(
				"apply override %s.%s: %w",
				override.section,
				override.key,
				err,
			)
		}
	}

	return nil
}
