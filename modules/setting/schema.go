package setting

import (
	"fmt"
	"strings"

	"github.com/datvtph41107/bdspro/modules/ipactive"
)

var knownConfigSections = map[string][]string{
	"": {"RUN_MODE", "WORK_PATH"},
	"server": {
		"HTTP_SERVER_ADDRESS", "PUBLIC_ORIGIN", "TRUSTED_PROXIES",
		"HTTP_SHUTDOWN_TIMEOUT", "GRACEFUL_HAMMER_TIMEOUT",
	},
	"database": {"DB_SOURCE", "MIGRATION_URL"},
	"redis":    {"ADDR"},
	"api":      {"MAX_RESPONSE_ITEMS", "DEFAULT_PAGING_NUM", "MAX_REQUEST_BODY_BYTES"},
	"log":      {"MODE", "FORMAT", "LEVEL", "ROOT_PATH"},
	"log.file": {"FILE_NAME"},
	"security": {"TOKEN_SYMMETRIC_KEY", "ACCESS_TOKEN_DURATION", "REFRESH_TOKEN_DURATION", "REMEMBER_SESSION_DURATION"},
	"cors":     {"ALLOWED_ORIGINS"},
	"mailer":   {"SMTP_ADDRESS", "FROM", "USERNAME", "PASSWORD"},
	"sms":      {"ACCOUNT_SID", "AUTH_TOKEN", "SENDER_ID"},
	"google":   {"CLIENT_ID", "CLIENT_SECRET", "CALLBACK_URL"},
	"ipactive": {},
}

func normalizeConfigSection(section string) string {
	if section == "" || strings.EqualFold(section, "DEFAULT") {
		return ""
	}
	return strings.ToLower(section)
}

func isKnownConfigKey(section, key string) bool {
	section = normalizeConfigSection(section)
	key = strings.ToUpper(key)
	if section == "ipactive" {
		for action := range ipactive.DefaultPolicies() {
			prefix := strings.ToUpper(string(action))
			if key == prefix+"_LIMIT" || key == prefix+"_WINDOW" || key == prefix+"_MODE" {
				return true
			}
		}
		return false
	}
	keys, exists := knownConfigSections[section]
	if !exists {
		return false
	}
	for _, allowed := range keys {
		if allowed == key {
			return true
		}
	}
	return false
}

func validateKnownKeys(cfg ConfigProvider) error {
	for _, section := range cfg.Sections() {
		name := normalizeConfigSection(section.Name())
		if _, ok := knownConfigSections[name]; !ok {
			return fmt.Errorf("unknown configuration section %q", section.Name())
		}
		for _, key := range section.Keys() {
			if !isKnownConfigKey(name, key.Name()) {
				return fmt.Errorf("unknown configuration key %q in section %q", key.Name(), section.Name())
			}
		}
	}
	return nil
}
