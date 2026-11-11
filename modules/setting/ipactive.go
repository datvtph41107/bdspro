package setting

import (
    "fmt"
    "strconv"
    "strings"
    "time"

    "github.com/datvtph41107/bdspro/modules/ipactive"
)

var IPActive struct { Policies map[ipactive.Action]ipactive.Policy }

func loadIPActiveFrom(rootCfg ConfigProvider) error {
    policies := ipactive.DefaultPolicies()
    sec := rootCfg.Section("ipactive")
    for action, policy := range policies {
        prefix := strings.ToUpper(string(action))
        if raw := sec.Key(prefix+"_LIMIT").String(); raw != "" {
            limit, err := strconv.ParseInt(raw, 10, 64)
            if err != nil { return fmt.Errorf("IPActive %s limit: %w", action, err) }
            policy.Limit = limit
        }
        if raw := sec.Key(prefix+"_WINDOW").String(); raw != "" {
            window, err := time.ParseDuration(raw)
            if err != nil { return fmt.Errorf("IPActive %s window: %w", action, err) }
            policy.Window = window
        }
        if raw := sec.Key(prefix+"_MODE").String(); raw != "" {
            policy.Mode = ipactive.Mode(strings.ToLower(raw))
        }
        if err := policy.Validate(); err != nil { return fmt.Errorf("IPActive %s: %w", action, err) }
        policies[action] = policy
    }
    IPActive.Policies = policies
    return nil
}
