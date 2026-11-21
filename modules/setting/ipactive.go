package setting

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/ipactive"
)

// IPActiveSettings là policy input của ingress Redis limiter.
type IPActiveSettings struct {
	Policies map[ipactive.Action]ipactive.Policy
}

var IPActive IPActiveSettings

// clone avoids sharing the mutable policy map with legacy global consumers.
func (s IPActiveSettings) clone() IPActiveSettings {
	policies := make(map[ipactive.Action]ipactive.Policy, len(s.Policies))
	for action, policy := range s.Policies {
		policies[action] = policy
	}
	return IPActiveSettings{Policies: policies}
}

func loadIPActiveFrom(cfg ConfigProvider) (IPActiveSettings, error) {
	policies := ipactive.DefaultPolicies()
	sec := cfg.Section("ipactive")
	for action, policy := range policies {
		prefix := strings.ToUpper(string(action))
		if raw := sec.Key(prefix + "_LIMIT").String(); raw != "" {
			limit, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return IPActiveSettings{}, fmt.Errorf("ipactive.%s_LIMIT: %w", prefix, err)
			}
			policy.Limit = limit
		}
		if raw := sec.Key(prefix + "_WINDOW").String(); raw != "" {
			window, err := time.ParseDuration(raw)
			if err != nil {
				return IPActiveSettings{}, fmt.Errorf("ipactive.%s_WINDOW: %w", prefix, err)
			}
			policy.Window = window
		}
		if raw := sec.Key(prefix + "_MODE").String(); raw != "" {
			policy.Mode = ipactive.Mode(strings.ToLower(raw))
		}
		if err := policy.Validate(); err != nil {
			return IPActiveSettings{}, fmt.Errorf("ipactive.%s: %w", prefix, err)
		}
		policies[action] = policy
	}
	return IPActiveSettings{Policies: policies}, nil
}
