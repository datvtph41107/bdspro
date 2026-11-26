package setting

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	EnvConfigKeyPrefix     = "BDS__"
	EnvConfigKeySuffixFile = "__FILE"
)

var envEscapedBytes = regexp.MustCompile(`_0[xX](?:[0-9a-fA-F]{2})+_`)

type configOverride struct {
	section string
	key     string
	value   string
}

func decodeEnvPart(encoded string) (string, error) {
	var b strings.Builder
	start := 0
	for _, loc := range envEscapedBytes.FindAllStringIndex(encoded, -1) {
		b.WriteString(encoded[start:loc[0]])
		part := encoded[loc[0]:loc[1]]
		decoded, err := hex.DecodeString(part[3 : len(part)-1])
		if err != nil {
			return "", fmt.Errorf("decode environment key: %w", err)
		}
		b.Write(decoded)
		start = loc[1]
	}
	b.WriteString(encoded[start:])
	if !utf8.ValidString(b.String()) {
		return "", errors.New("environment name contains invalid UTF-8")
	}
	return b.String(), nil
}

func decodeEnvironmentKey(name string) (section, key string, fromFile, ok bool) {
	if !strings.HasPrefix(name, EnvConfigKeyPrefix) {
		return "", "", false, false
	}
	encoded := strings.TrimPrefix(name, EnvConfigKeyPrefix)
	if strings.HasSuffix(encoded, EnvConfigKeySuffixFile) {
		fromFile = true
		encoded = strings.TrimSuffix(encoded, EnvConfigKeySuffixFile)
	}
	rawSection, rawKey, found := strings.Cut(encoded, "__")
	if !found || rawKey == "" || strings.Contains(rawKey, "__") {
		return "", "", fromFile, false
	}
	var err error
	section, err = decodeEnvPart(rawSection)
	if err != nil {
		return "", "", fromFile, false
	}
	key, err = decodeEnvPart(rawKey)
	if err != nil || key == "" {
		return "", "", fromFile, false
	}
	return strings.ToLower(section), key, fromFile, true
}

const maxSecretFileBytes int64 = 64 * 1024

func readEnvironmentFile(path string) (string, error) {
	if path == "" {
		return "", errors.New("secret file path is empty")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open secret file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxSecretFileBytes+1))
	if err != nil {
		return "", fmt.Errorf("read secret file: %w", err)
	}
	if int64(len(content)) > maxSecretFileBytes {
		return "", errors.New("secret file exceeds size limit")
	}
	switch {
	case bytes.HasSuffix(content, []byte("\r\n")):
		content = content[:len(content)-2]
	case bytes.HasSuffix(content, []byte("\n")):
		content = content[:len(content)-1]
	}
	if !utf8.Valid(content) {
		return "", errors.New("secret file is not valid UTF-8")
	}
	return string(content), nil
}

func collectEnvironmentOverrides(envs []string) ([]configOverride, error) {
	overrides := make([]configOverride, 0)
	seen := make(map[string]struct{})
	for _, entry := range envs {
		name, value, found := strings.Cut(entry, "=")
		if !found || !strings.HasPrefix(name, EnvConfigKeyPrefix) {
			continue
		}
		section, key, fromFile, ok := decodeEnvironmentKey(name)
		if !ok {
			return nil, fmt.Errorf("invalid config environment name %q", name)
		}
		if !isKnownConfigKey(section, key) {
			return nil, fmt.Errorf("unknown config environment %q", name)
		}

		switch strings.ToLower(section) {
		case "audit", "oauth2", "phone", "queue":
			return nil, fmt.Errorf("environment %q targets removed config section %q", name, section)
		}
		identity := strings.ToLower(section) + "\x00" + strings.ToUpper(key)
		if _, ok := seen[identity]; ok {
			return nil, fmt.Errorf("duplicate configuration override %q", name)
		}
		seen[identity] = struct{}{}
		if fromFile {
			secret, err := readEnvironmentFile(value)
			if err != nil {
				return nil, fmt.Errorf("environment %s: %w", name, err)
			}
			value = secret
		}
		overrides = append(overrides, configOverride{section: section, key: key, value: value})
	}
	return overrides, nil
}

func applyEnvironmentOverrides(provider ConfigProvider, envs []string) error {
	p, ok := provider.(*iniConfigProvider)
	if !ok || p == nil {
		return errors.New("configuration provider does not support overrides")
	}
	overrides, err := collectEnvironmentOverrides(envs)
	if err != nil {
		return err
	}
	for _, item := range overrides {
		if err := p.set(item.section, item.key, item.value); err != nil {
			return fmt.Errorf("apply override %s.%s: %w", item.section, item.key, err)
		}
	}
	return nil
}
