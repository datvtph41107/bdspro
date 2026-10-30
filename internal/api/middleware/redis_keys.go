package middleware

// Redis namespaces are persisted operational state. Changing these bytes
// resets counters and requires a coordinated key migration.
const (
	securityDeniedKeyPrefix = "bdspro:security:denied:v1:"
	sourceLimitKeyPrefix    = "bdspro:source_limit:v1:"
	authThrottleKeyPrefix   = "bdspro:auth_limit:v1:"
)
