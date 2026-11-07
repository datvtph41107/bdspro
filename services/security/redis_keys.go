package security

import (
	"crypto/sha256"
	"encoding/hex"
)

// Redis namespaces are persisted operational state. Changing these bytes
// resets counters and requires a coordinated key migration.
const (
	securityDeniedKeyPrefix = "bdspro:security:denied:v1:"
	authThrottleKeyPrefix   = "bdspro:auth_limit:v1:"
)

// hashedRateKey preserves the existing one-way identifier used by denial counters.
func hashedRateKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
