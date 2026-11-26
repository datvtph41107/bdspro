package ipactive

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"

	"github.com/datvtph41107/bdspro/modules/timeutil"
	redis "github.com/go-redis/redis/v8"
)

const SourceKeyPrefix = "bdspro:source_limit:v1:"

var (
	ErrUnavailable   = errors.New("IPActive admission unavailable")
	ErrUnknownAction = errors.New("unknown IPActive action")
	ErrInvalidSource = errors.New("invalid IPActive source")
)

var countScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
if n > tonumber(ARGV[2]) then
    local ttl = redis.call('TTL', KEYS[1])
    if ttl < 1 then ttl = 1 end
    return {n, ttl}
end
return {n, 0}
`)

var cooldownScript = redis.NewScript(`
if redis.call('SET', KEYS[1], '1', 'EX', ARGV[1], 'NX') then
    return {1, 0}
end
local ttl = redis.call('TTL', KEYS[1])
if ttl < 1 then ttl = 1 end
return {0, ttl}
`)

type Limiter struct {
	client   *redis.Client
	policies map[Action]Policy
}

func NewRedis(client *redis.Client, policies map[Action]Policy) (*Limiter, error) {
	if client == nil {
		return nil, ErrUnavailable
	}
	if len(policies) == 0 {
		return nil, errors.New("IPActive requires at least one policy")
	}
	copied := make(map[Action]Policy, len(policies))
	for action, policy := range policies {
		if !validAction(action) {
			return nil, fmt.Errorf("invalid IPActive action %q", action)
		}
		if err := policy.Validate(); err != nil {
			return nil, fmt.Errorf("IPActive %s: %w", action, err)
		}
		copied[action] = policy
	}
	return &Limiter{client: client, policies: copied}, nil
}

func (l *Limiter) Check(ctx context.Context, action Action, sourceIP string) (Event, error) {
	if l == nil || l.client == nil {
		return Event{}, ErrUnavailable
	}
	policy, ok := l.policies[action]
	if !ok {
		return Event{}, ErrUnknownAction
	}
	if _, err := netip.ParseAddr(sourceIP); err != nil {
		return Event{}, ErrInvalidSource
	}
	digest := sha256.Sum256([]byte(sourceIP))
	sourceID := hex.EncodeToString(digest[:])
	return l.checkKey(ctx, action, policy, sourceID, SourceKeyPrefix)
}

func (l *Limiter) CheckSubject(ctx context.Context, action Action, subjectID string) (Event, error) {
	if l == nil || l.client == nil {
		return Event{}, ErrUnavailable
	}
	policy, ok := l.policies[action]
	if !ok {
		return Event{}, ErrUnknownAction
	}
	decoded, err := hex.DecodeString(subjectID)
	if err != nil || len(decoded) != sha256.Size {
		return Event{}, ErrInvalidSource
	}
	return l.checkKey(ctx, action, policy, hex.EncodeToString(decoded), "bdspro:subject_limit:v1:")
}

func (l *Limiter) checkKey(ctx context.Context, action Action, policy Policy, sourceID, prefix string) (Event, error) {
	key := prefix + string(action) + ":" + sourceID
	seconds := timeutil.SecondsCeil(policy.Window)

	script := countScript
	args := []interface{}{seconds, policy.Limit}
	if policy.Mode == ModeCooldown {
		script = cooldownScript
		args = args[:1]
	}
	values, err := script.Run(ctx, l.client, []string{key}, args...).Slice()
	if err != nil {
		return Event{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if len(values) != 2 {
		return Event{}, fmt.Errorf("%w: invalid Redis reply", ErrUnavailable)
	}
	attempts, attemptsOK := values[0].(int64)
	ttl, ttlOK := values[1].(int64)
	if !attemptsOK || !ttlOK || attempts < 0 || ttl < 0 {
		return Event{}, fmt.Errorf("%w: invalid Redis reply", ErrUnavailable)
	}
	allowed := attempts <= policy.Limit
	if policy.Mode == ModeCooldown {
		allowed = attempts == 1
	}
	return Event{
		Action: action, SourceID: sourceID, Mode: policy.Mode,
		Allowed: allowed, Attempts: attempts, Limit: policy.Limit,
		RetryAfter: ttl, OccurredAt: timeutil.TimeStampNow(),
	}, nil
}
