package org

import (
	"context"
	"errors"
)

type inviteSigningKey struct{}

var errInviteSigningKeyUnavailable = errors.New("org: invitation signing key unavailable")

// WithInviteSigningKey binds an optional process-configured secret to this
// Server's requests. Missing configuration is rejected by the issuing service,
// not by unrelated routes, so health and token-only deployments remain usable.
func WithInviteSigningKey(ctx context.Context, key string) context.Context {
	if ctx == nil {
		panic("org: nil request context")
	}
	if key == "" {
		return ctx
	}
	return context.WithValue(ctx, inviteSigningKey{}, key)
}

func inviteSigningKeyFromContext(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", errInviteSigningKeyUnavailable
	}
	key, ok := ctx.Value(inviteSigningKey{}).(string)
	if !ok || key == "" {
		return "", errInviteSigningKeyUnavailable
	}
	return key, nil
}
