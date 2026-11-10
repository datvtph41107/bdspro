package org

import "context"

type inviteSigningKey struct{}

// The org service owns the invitation proof key, scoped to its Server.
func WithInviteSigningKey(ctx context.Context,key string) context.Context {
	if ctx==nil||key=="" {panic("org: missing invite signing key")}
	return context.WithValue(ctx,inviteSigningKey{},key)
}
func InviteSigningKeyFromContext(ctx context.Context) string {
	if ctx==nil {panic("org: nil context")}
	key,ok:=ctx.Value(inviteSigningKey{}).(string)
	if !ok||key=="" {panic("org: invite signing key not initialized")}
	return key
}
