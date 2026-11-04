package setting

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/datvtph41107/bdspro/modules/ipactive"
)

// LoadIPActivePolicies reads optional [ipactive] ACTION_LIMIT,
// ACTION_WINDOW and ACTION_MODE overrides. Unknown actions cannot be added
// through INI: every route must explicitly declare a supported action.
func LoadIPActivePolicies(provider *ConfigProvider) (map[ipactive.Action]ipactive.Policy, error) {
	if provider == nil {
		return nil, fmt.Errorf("IPActive configuration provider is nil")
	}
	policies := ipactive.DefaultPolicies()
	for action, policy := range policies {
		prefix := strings.ToUpper(string(action))
		if raw := provider.String("ipactive", prefix+"_LIMIT"); raw != "" {
			limit, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("IPActive %s limit: %w", action, err)
			}
			policy.Limit = limit
		}
		if raw := provider.String("ipactive", prefix+"_WINDOW"); raw != "" {
			window, err := time.ParseDuration(raw)
			if err != nil {
				return nil, fmt.Errorf("IPActive %s window: %w", action, err)
			}
			policy.Window = window
		}
		if raw := provider.String("ipactive", prefix+"_MODE"); raw != "" {
			policy.Mode = ipactive.Mode(strings.ToLower(raw))
		}
		if err := policy.Validate(); err != nil {
			return nil, fmt.Errorf("IPActive %s: %w", action, err)
		}
		policies[action] = policy
	}
	return policies, nil
}
